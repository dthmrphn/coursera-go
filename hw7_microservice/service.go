package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
)

type bizServer struct {
	UnimplementedBizServer
}

func (s *bizServer) Check(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s *bizServer) Add(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

func (s *bizServer) Test(context.Context, *Nothing) (*Nothing, error) {
	return &Nothing{}, nil
}

type admServer struct {
	UnimplementedAdminServer
}

func (s *admServer) Logging(*Nothing, Admin_LoggingServer) error {
	return nil
}

func (s *admServer) Statistics(*StatInterval, Admin_StatisticsServer) error {
	return nil
}

type ACL map[string][]string

func ParseACL(acl string) (ACL, error) {
	m := ACL{}
	err := json.Unmarshal([]byte(acl), &m)
	if err != nil {
		return nil, fmt.Errorf("acl syntax error")
	}

	return m, nil
}

func match(allowed, method string) bool {
	allowed = strings.Split(allowed, "/")[2]
	method = strings.Split(method, "/")[2]

	if allowed == "*" || allowed == method {
		return true
	}

	return false
}

func (acl ACL) Allowed(consumer string, method string) bool {
	methods, ok := acl[consumer]
	if !ok {
		return ok
	}

	for _, m := range methods {
		if match(m, method) {
			return true
		}
	}

	return false
}

type Interceptor struct {
	acl ACL
}

func NewInterceptor(acl ACL) *Interceptor {
	return &Interceptor{acl}
}

func (i *Interceptor) unary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "missing incoming metadata")
	}

	consumers, ok := md["consumer"]
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unknown consumer")
	}

	if !i.acl.Allowed(consumers[0], info.FullMethod) {
		return nil, status.Errorf(codes.Unauthenticated, "method %s isn't allowed for %s", info.FullMethod, consumers[0])
	}

	return handler(ctx, req)
}

func (i *Interceptor) stream(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	md, ok := metadata.FromIncomingContext(ss.Context())
	if !ok {
		return status.Errorf(codes.Internal, "missing incoming metadata")
	}

	consumers, ok := md["consumer"]
	if !ok {
		return status.Errorf(codes.Unauthenticated, "unknown consumer")
	}

	if !i.acl.Allowed(consumers[0], info.FullMethod) {
		return status.Errorf(codes.Unauthenticated, "method %s isn't allowed for %s", info.FullMethod, consumers[0])
	}

	return handler(srv, ss)
}

type logger struct {
	name string
}

func (l *logger) unary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	fmt.Println("unary", l.name)
	return handler(ctx, req)
}

type auth struct {
	name string
}

func (l *auth) unary(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	fmt.Println("unary", l.name)
	return handler(ctx, req)
}

func StartMyMicroservice(ctx context.Context, addr, acl string) error {
	macl, err := ParseACL(acl)
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	iauth := &auth{"auth int"}
	ilogg := &logger{"log int"}

	icp := NewInterceptor(macl)

	opts := []grpc.ServerOption{
		// grpc.UnaryInterceptor(icp.unary),
		grpc.StreamInterceptor(icp.stream),
		grpc.ChainUnaryInterceptor(
		    iauth.unary,
		    ilogg.unary,
		),
	}

	srv := grpc.NewServer(opts...)
	RegisterAdminServer(srv, &admServer{})
	RegisterBizServer(srv, &bizServer{})

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
	}()

	go srv.Serve(lis)

	return nil
}
