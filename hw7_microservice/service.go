package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"strings"
	"sync"
	"time"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
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

	subs *eventSubs
}

func (s *admServer) Logging(n *Nothing, srv Admin_LoggingServer) error {
	id, events := s.subs.newSub()
	defer s.subs.deleteSub(id)

	for e := range events {
		if err := srv.Send(e); err != nil {
			return err
		}
	}

	return nil
}

func (s *admServer) Statistics(i *StatInterval, srv Admin_StatisticsServer) error {
	id, events := s.subs.newSub()
	defer s.subs.deleteSub(id)

	stat := &Stat{
		ByMethod:   map[string]uint64{},
		ByConsumer: map[string]uint64{},
	}

	tick := time.NewTimer(time.Duration(i.IntervalSeconds) * time.Second)
	defer tick.Stop()

	for {
		select {
		case e, ok := <-events:
			if ok {
				stat.ByConsumer[e.Consumer]++
				stat.ByMethod[e.Method]++
			} else {
				return nil
			}

		case <-tick.C:
			stat.Timestamp = time.Now().Unix()
			printshit(stat)
			if err := srv.Send(stat); err != nil {
				return err
			}
			stat = &Stat{
				ByMethod:   map[string]uint64{},
				ByConsumer: map[string]uint64{},
			}
		}
	}
}

func printshit(s *Stat) {
	fmt.Printf("to send: \n")
	for k, v := range s.ByConsumer {
		fmt.Println(k, v)
	}
	for k, v := range s.ByMethod {
		fmt.Println(k, v)
	}
}

type eventSubs struct {
	id   int
	subs map[int]chan *Event
	mux  *sync.RWMutex
}

func newEventSubs() *eventSubs {
	return &eventSubs{
		subs: map[int]chan *Event{},
		mux:  &sync.RWMutex{},
	}
}

func (s *eventSubs) newSub() (int, chan *Event) {
	s.mux.Lock()
	defer s.mux.Unlock()

	s.id++
	s.subs[s.id] = make(chan *Event)
	return s.id, s.subs[s.id]
}

func (s *eventSubs) deleteSub(id int) {
	s.mux.Lock()
	defer s.mux.Unlock()

	if sub, ok := s.subs[id]; ok {
		close(sub)
		delete(s.subs, id)
	}
}

func (s *eventSubs) delete() {
	for id := range s.subs {
		s.deleteSub(id)
	}
}

func (s *eventSubs) notify(e *Event) {
	s.mux.RLock()
	defer s.mux.RUnlock()

	for _, sub := range s.subs {
		sub <- e
	}
}

type aclAuth map[string][]string

func ParseACL(acl string) (aclAuth, error) {
	m := aclAuth{}
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

func (acl aclAuth) allowed(consumer string, method string) bool {
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

type wrappedStream struct {
	grpc.ServerStream
	ctx context.Context
}

func (s *wrappedStream) Context() context.Context {
	return s.ctx
}

type serviceCtxKey struct{}

func newCtxEventFromMD(ctx context.Context, method string) (context.Context, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if !ok {
		return nil, status.Errorf(codes.Internal, "missing incoming metadata")
	}

	consumers, ok := md["consumer"]
	if !ok {
		return nil, status.Errorf(codes.Unauthenticated, "unknown consumer")
	}

	host := ""
	if pr, ok := peer.FromContext(ctx); ok {
		host = pr.Addr.String()
	}

	evt := &Event{
		Host:      host,
		Method:    method,
		Consumer:  consumers[0],
		Timestamp: time.Now().Unix(),
	}

	return context.WithValue(ctx, serviceCtxKey{}, evt), nil
}

func eventFromCtx(ctx context.Context) (*Event, error) {
	e, ok := ctx.Value(serviceCtxKey{}).(*Event)
	if !ok {
		return nil, status.Errorf(codes.Internal, "missing event in context")
	}

	return e, nil
}

func MetaUnaryInterceptor() grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		if newCtx, err := newCtxEventFromMD(ctx, info.FullMethod); err != nil {
			return nil, err
		} else {
			return handler(newCtx, req)
		}
	}
}

func MetaStreamInterceptor() grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		if newCtx, err := newCtxEventFromMD(ss.Context(), info.FullMethod); err != nil {
			return err
		} else {
			return handler(srv, &wrappedStream{ss, newCtx})
		}
	}
}

func AuthUnaryInterceptor(acl aclAuth) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		event, _ := eventFromCtx(ctx)
		if !acl.allowed(event.Consumer, event.Method) {
			return nil, status.Errorf(codes.Unauthenticated, "method %s isnt allowed for %s", event.Method, event.Consumer)
		}
		return handler(ctx, req)
	}
}

func AuthStreamInterceptor(acl aclAuth) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		event, _ := eventFromCtx(ss.Context())
		if !acl.allowed(event.Consumer, event.Method) {
			return status.Errorf(codes.Unauthenticated, "method %s isnt allowed for %s", event.Method, event.Consumer)
		}
		return handler(srv, ss)
	}
}

func LogUnaryInterceptor(events *eventSubs) grpc.UnaryServerInterceptor {
	return func(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
		event, _ := eventFromCtx(ctx)
		events.notify(event)
		return handler(ctx, req)
	}
}

func LogStreamInterceptor(events *eventSubs) grpc.StreamServerInterceptor {
	return func(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
		event, _ := eventFromCtx(ss.Context())
		events.notify(event)
		return handler(srv, ss)
	}
}

func StartMyMicroservice(ctx context.Context, addr, acl string) error {
	auth, err := ParseACL(acl)
	if err != nil {
		return err
	}

	lis, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	subs := newEventSubs()

	opts := []grpc.ServerOption{
		grpc.ChainUnaryInterceptor(
			MetaUnaryInterceptor(),
			AuthUnaryInterceptor(auth),
			LogUnaryInterceptor(subs),
		),
		grpc.ChainStreamInterceptor(
			MetaStreamInterceptor(),
			AuthStreamInterceptor(auth),
			LogStreamInterceptor(subs),
		),
	}

	srv := grpc.NewServer(opts...)
	RegisterAdminServer(srv, &admServer{subs: subs})
	RegisterBizServer(srv, &bizServer{})

	go func() {
		<-ctx.Done()
		srv.GracefulStop()
		subs.delete()
	}()

	go srv.Serve(lis)

	return nil
}
