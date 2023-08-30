package main

import (
	"encoding/json"
	"encoding/xml"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
)

var (
	ErrorMissedField    = fmt.Errorf("field missed")
	ErrorStrIntCast     = fmt.Errorf("couldnt cast s to i")
	ErrorWrongValue     = fmt.Errorf("wrong value")
	ErrorWrongQuery     = fmt.Errorf("query is wrong")
	ErrorInvalidOrder   = fmt.Errorf("ErrorBadOrderField")
	ErrorInvalidOrderby = fmt.Errorf("should be [-1:1]")
)

type root struct {
	Name xml.Name `xml:"root"`
	Rows []row    `xml:"row"`
}

type row struct {
	Id        int    `xml:"id"`
	FirstName string `xml:"first_name"`
	LastName  string `xml:"last_name"`
	Age       int    `xml:"age"`
	About     string `xml:"about"`
	Gender    string `xml:"gender"`
}

func (r *row) convert() User {
	return User{
		Id:     r.Id,
		Age:    r.Age,
		About:  r.About,
		Gender: r.Gender,
		Name:   r.FirstName + " " + r.LastName,
	}
}

type server struct {
	token string
	users []User
}

func NewServer(token, fp string) (*server, error) {
	data, err := os.ReadFile(fp)
	if err != nil {
		return nil, err
	}

	root := &root{}
	err = xml.Unmarshal(data, root)
	if err != nil {
		return nil, err
	}

	users := make([]User, len(root.Rows))
	for i, row := range root.Rows {
		users[i] = row.convert()
	}

	return &server{
		token: token,
		users: users,
	}, nil
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s.token != r.Header.Get("AccessToken") {
		w.WriteHeader(http.StatusUnauthorized)
		return
	}

	sr, err := s.SearchRequest(r.URL.RawQuery)
	if err != nil {
		j, _ := json.Marshal(SearchErrorResponse{Error: err.Error()})
		w.WriteHeader(http.StatusBadRequest)
		w.Write(j)
		return
	}

	users, code := s.SearchUsers(sr)

	w.WriteHeader(code)
	w.Write(users)
}

func value(q url.Values, key string) (int, error) {
	if q.Get(key) == "" {
		return 0, fmt.Errorf("%s: %w", key, ErrorMissedField)
	}

	v, err := strconv.Atoi(q.Get(key))
	if err != nil {
		return 0, fmt.Errorf("%s: %w", key, ErrorStrIntCast)
	}

	return v, nil
}

func (s *server) SearchRequest(req string) (*SearchRequest, error) {
	q, err := url.ParseQuery(req)
	if err != nil {
		return nil, fmt.Errorf("could not parse query: %w", ErrorWrongQuery)
	}

	limit, err := value(q, "limit")
	if err != nil {
		return nil, err
	}

	offset, err := value(q, "offset")
	if err != nil {
		return nil, err
	}

	orderBy, err := value(q, "order_by")
	if err != nil {
		return nil, err
	}

	if limit > 25 {
		limit = 25
	}

	if limit < 0 {
		return nil, fmt.Errorf("limit is negative: %w", ErrorWrongValue)
	}

	if offset < 0 {
		return nil, fmt.Errorf("offset is negative: %w", ErrorWrongValue)
	}

	if offset > len(s.users) {
		return nil, fmt.Errorf("offset is too big: %w", ErrorWrongValue)
	}

	switch orderBy {
	case OrderByAsIs:
	case OrderByAsc:
	case OrderByDesc:
	default:
		return nil, ErrorInvalidOrderby
	}

	query := q.Get("query")
	order := q.Get("order_field")

	switch order {
	case "":
	case "Id":
	case "Age":
	case "Name":
	default:
		return nil, ErrorInvalidOrder
	}

	return &SearchRequest{
		Limit:      limit,
		Offset:     offset,
		Query:      query,
		OrderField: order,
		OrderBy:    orderBy,
	}, nil
}

func sortUsers(u []User, order string, inc int) []User {
	if inc == OrderByAsIs {
		return u
	}

	id := func(a, b User) bool {
		if inc > 0 {
			return a.Id < b.Id
		}
		return a.Id > b.Id
	}

	age := func(a, b User) bool {
		if inc > 0 {
			return a.Age < b.Age
		}
		return a.Age > b.Age
	}

	name := func(a, b User) bool {
		if inc > 0 {
			return a.Name < b.Name
		}
		return a.Name > b.Name
	}

	switch order {
	case "Id":
		sort.Slice(u, func(i, j int) bool { return id(u[i], u[j]) })
	case "Age":
		sort.Slice(u, func(i, j int) bool { return age(u[i], u[j]) })
	case "":
		fallthrough
	case "Name":
		sort.Slice(u, func(i, j int) bool { return name(u[i], u[j]) })
	}

	return u
}

func (s *server) SearchUsers(sr *SearchRequest) ([]byte, int) {
	rv := make([]User, 0)

	for i := sr.Offset; i < sr.Limit; i++ {
		if !strings.Contains(s.users[i].Name, sr.Query) && !strings.Contains(s.users[i].About, sr.Query) {
			continue
		}
		rv = append(rv, s.users[i])
	}

	rv = sortUsers(rv, sr.OrderField, sr.OrderBy)
	js, _ := json.Marshal(rv)

	return js, http.StatusOK
}
