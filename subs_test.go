package subs

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"

	"github.com/google/uuid"
)

type MockDB struct {
	db map[string]Sub
}

func (m *MockDB) Create(sub Sub) (string, error) {
	id := uuid.NewString()
	sub.ID = id
	m.db[id] = sub

	return id, nil
}

func (m *MockDB) Read(id string) (Sub, error) {

	sub, exists := m.db[id]
	if !exists {
		return Sub{}, fmt.Errorf("not found")
	}

	return sub, nil
}

func (m *MockDB) Update(id string, sub Sub) error {
	if _, exists := m.db[id]; !exists {
		return fmt.Errorf("not found")
	}
	m.db[id] = sub

	return nil
}

func (m *MockDB) Delete(id string) error {

	if _, ok := m.db[id]; !ok {
		return fmt.Errorf("not found")
	}
	delete(m.db, id)

	return nil
}

func (m *MockDB) List() ([]Sub, error) {

	if len(m.db) == 0 {
		return []Sub{}, fmt.Errorf("not found")
	}

	var subs []Sub
	for _, sub := range m.db {
		subs = append(subs, sub)
	}

	return subs, nil
}

func (m *MockDB) Sum(filter Sub) (int, error) {

	var sum int
	s := strings.Split(filter.Start, "-")
	e := strings.Split(*filter.End, "-")
	u := filter.User_ID
	n := filter.Service

	for _, sub := range m.db {

		if u != "" && sub.User_ID != u {
			continue
		}

		if n != "" && sub.Service != n {
			continue
		}

		ss := strings.Split(sub.Start, "-")

		m1, _ := strconv.Atoi(s[0])
		m3, _ := strconv.Atoi(e[0])
		m2, _ := strconv.Atoi(ss[0])

		if m1 >= m2 {
			if sub.End != nil {
				ee := strings.Split(*sub.End, "-")
				m4, _ := strconv.Atoi(ee[0])
				if m3 >= m4 {
					sum += sub.Price * (m4 - m1 + 1)
					continue
				}
			}
			sum += sub.Price * (m3 - m1 + 1)
		} else if m2 <= m3 {
			if sub.End != nil {
				ee := strings.Split(*sub.End, "-")
				m4, _ := strconv.Atoi(ee[0])
				if m3 >= m4 {
					sum += sub.Price * (m4 - m2 + 1)
					continue
				}
			}
			sum += sub.Price * (m3 - m2 + 1)
		}
	}

	return sum, nil
}

func compareSubs(t *testing.T, s1 Sub, s2 Sub) {

	t.Helper()

	if s1.Service != s2.Service {
		t.Fatal("service not equal")
	}
	if s1.Price != s2.Price {
		t.Fatal("price not equal")
	}
	if s1.User_ID != s2.User_ID {
		t.Fatal("user not equal")
	}
	if s1.Start != s2.Start {
		t.Fatal("start not equal")
	}
	if (s1.End == nil && s2.End != nil) || (s1.End != nil && s2.End == nil) || (s1.End != nil && s2.End != nil && *s1.End != *s2.End) {
		t.Fatal("end not equal")
	}
}

func TestValidateSub(t *testing.T) {

	s := Sub{
		Service: "service",
		Price:   400,
		User_ID: uuid.NewString(),
		Start:   "07-2024",
		End:     func() *string { s := "10-2024"; return &s }(),
	}

	if err := validateSub(s); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	s2 := s
	s2.End = nil
	if err := validateSub(s2); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	s3 := s
	s3.Start = "123"
	if err := validateSub(s3); err == nil {
		t.Error("expected err, got nil")

	}

	s4 := s
	s4.User_ID = "123"
	if err := validateSub(s4); err == nil {
		t.Errorf("expected err, got nil")
	}

	s5 := s
	s5.Price = -2
	if err := validateSub(s5); err == nil {
		t.Errorf("expected err, got nil")
	}

	s6 := s
	s6.Service = ""
	if err := validateSub(s6); err == nil {
		t.Errorf("expected err, got nil")
	}
}

func TestValidateFilter(t *testing.T) {

	f := Sub{
		Start: "07-2024",
		End:   func() *string { s := "10-2024"; return &s }(),
	}

	if err := validateFilter(f); err != nil {
		t.Errorf("expected nil, got %v", err)
	}

	f2 := f
	f2.End = nil
	if err := validateFilter(f2); err == nil {
		t.Error("expected err, got nil")
	}
}

func testCreatePayload(t *testing.T, server_url string, s Sub) string {

	body, _ := json.Marshal(s)
	resp, err := http.Post(server_url+"/subs", "application/json", bytes.NewBuffer(body))
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 201 {
		t.Errorf("expected status: 201, got: %v", resp.StatusCode)
	}

	var r struct {
		Sub_id string `json:"sub_id"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Error(err)
	}

	if err := uuid.Validate(r.Sub_id); err != nil {
		t.Error(err)
	}

	return r.Sub_id
}

func TestCreateHandler(t *testing.T) {

	var m MockDB
	m.db = make(map[string]Sub)
	db = &m

	server := httptest.NewServer(newRouter())
	defer server.Close()

	s := Sub{
		Service: "service",
		Price:   400,
		User_ID: uuid.NewString(),
		Start:   "07-2024",
	}

	t.Run("empty", func(t *testing.T) {
		resp, err := http.Post(server.URL+"/subs", "application/json", bytes.NewBuffer([]byte{}))
		if err != nil {
			t.Error(err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("expected status: 400, got %v", resp.StatusCode)
		}
	})

	t.Run("no user id", func(t *testing.T) {
		s2 := s
		s2.User_ID = ""
		body, _ := json.Marshal(s2)
		resp, err := http.Post(server.URL+"/subs", "application/json", bytes.NewBuffer(body))
		if err != nil {
			t.Error(err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("expected status: 400, got %v", resp.StatusCode)
		}
	})

	t.Run("payload", func(t *testing.T) {
		id := testCreatePayload(t, server.URL, s)

		v, ok := m.db[id]
		if !ok {
			t.Error("not found in db")
		}

		compareSubs(t, v, s)
	})
}

func testReadPayload(t *testing.T, server_url string, sub_id string) Sub {

	resp, err := http.Get(server_url + "/subs/" + sub_id)
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected status: 200, got: %v", resp.StatusCode)
	}

	var r Sub
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Error(err)
	}

	if r.ID != sub_id {
		t.Errorf("expected id: %v, got: %v", sub_id, r.ID)
	}

	return r
}

func TestReadHandler(t *testing.T) {

	var m MockDB
	m.db = make(map[string]Sub)
	db = &m

	server := httptest.NewServer(newRouter())
	defer server.Close()

	t.Run("malformed id", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/subs/" + "123")
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Errorf("expected status: 400, got %v", resp.StatusCode)
		}
	})

	t.Run("wrong id", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/subs/" + uuid.NewString())
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 500 {
			t.Errorf("expected status: 500, got %v", resp.StatusCode)
		}
	})

	t.Run("payload", func(t *testing.T) {
		id := uuid.NewString()
		s := Sub{
			ID:      id,
			Service: "service",
			Price:   400,
			User_ID: uuid.NewString(),
			Start:   "07-2024",
		}
		m.db[id] = s

		compareSubs(t, s, testReadPayload(t, server.URL, id))
	})
}

func testUpdatePayload(t *testing.T, server_url string, sub_id string, s Sub) {

	body, _ := json.Marshal(s)
	req, err := http.NewRequest("PUT", server_url+"/subs/"+sub_id, bytes.NewBuffer(body))
	if err != nil {
		t.Error(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		t.Errorf("expected status: 200, got: %v", resp.StatusCode)
	}
}

func TestUpdateHandler(t *testing.T) {

	var m MockDB
	m.db = make(map[string]Sub)
	db = &m

	server := httptest.NewServer(newRouter())
	defer server.Close()

	t.Run("malformed id", func(t *testing.T) {
		req, err := http.NewRequest("PUT", server.URL+"/subs/"+"123", bytes.NewBuffer([]byte{}))
		if err != nil {
			t.Error(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Errorf("expected status: 400, got %v", resp.StatusCode)
		}
	})

	id := uuid.NewString()
	s := Sub{
		ID:      id,
		Service: "service",
		Price:   400,
		User_ID: uuid.NewString(),
		Start:   "07-2024",
	}
	m.db[id] = s

	s2 := Sub{
		ID:      id,
		Service: "service_2",
		Price:   700,
		User_ID: s.User_ID,
		Start:   "07-2024",
	}

	t.Run("empty", func(t *testing.T) {
		req, err := http.NewRequest("PUT", server.URL+"/subs/"+id, bytes.NewBuffer([]byte{}))
		if err != nil {
			t.Error(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Errorf("expected status: 400, got %v", resp.StatusCode)
		}
	})
	t.Run("wrong id", func(t *testing.T) {
		body, _ := json.Marshal(s2)

		req, err := http.NewRequest("PUT", server.URL+"/subs/"+uuid.NewString(), bytes.NewBuffer(body))
		if err != nil {
			t.Error(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 500 {
			t.Errorf("expected status: 500, got %v", resp.StatusCode)
		}
	})
	t.Run("no user id", func(t *testing.T) {

		s3 := s2
		s3.User_ID = ""
		body, _ := json.Marshal(s3)

		req, err := http.NewRequest("PUT", server.URL+"/subs/"+id, bytes.NewBuffer(body))
		if err != nil {
			t.Error(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Errorf("expected status: 400, got %v", resp.StatusCode)
		}
	})

	t.Run("payload", func(t *testing.T) {
		testUpdatePayload(t, server.URL, id, s2)
		compareSubs(t, m.db[id], s2)
	})
}

func testDeletePayload(t *testing.T, server_url string, sub_id string) {

	req, err := http.NewRequest("DELETE", server_url+"/subs/"+sub_id, bytes.NewBuffer([]byte{}))
	if err != nil {
		t.Error(err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 204 {
		t.Errorf("expected status: 204, got %v", resp.StatusCode)
	}
}

func TestDeleteHandler(t *testing.T) {

	var m MockDB
	m.db = make(map[string]Sub)
	db = &m

	server := httptest.NewServer(newRouter())
	defer server.Close()

	t.Run("malformed id", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", server.URL+"/subs/"+"123", bytes.NewBuffer([]byte{}))
		if err != nil {
			t.Error(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 400 {
			t.Errorf("expected status: 400, got %v", resp.StatusCode)
		}
	})
	t.Run("wrong id", func(t *testing.T) {
		req, err := http.NewRequest("DELETE", server.URL+"/subs/"+uuid.NewString(), bytes.NewBuffer([]byte{}))
		if err != nil {
			t.Error(err)
		}

		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Error(err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != 500 {
			t.Errorf("expected status: 500, got %v", resp.StatusCode)
		}
	})

	id := uuid.NewString()
	s := Sub{
		ID:      id,
		Service: "service",
		Price:   400,
		User_ID: uuid.NewString(),
		Start:   "07-2024",
	}
	m.db[id] = s

	t.Run("payload", func(t *testing.T) {
		testDeletePayload(t, server.URL, id)
		if _, ok := m.db[id]; ok {
			t.Error("not deleted from db")
		}
	})
}

func testListPayload(t *testing.T, server_url string) []Sub {

	resp, err := http.Get(server_url + "/subs")
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status: 200, got: %v", resp.StatusCode)
	}

	var subs []Sub
	if err := json.NewDecoder(resp.Body).Decode(&subs); err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	return subs
}

func TestListHandler(t *testing.T) {

	var m MockDB
	m.db = make(map[string]Sub)
	db = &m

	server := httptest.NewServer(newRouter())
	defer server.Close()

	t.Run("empty db", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/subs")
		if err != nil {
			t.Error(err)
		}

		if resp.StatusCode != 500 {
			t.Errorf("expected status: 500, got: %v", resp.StatusCode)
		}
	})

	id := uuid.NewString()
	s := Sub{
		ID:      id,
		Service: "service",
		Price:   400,
		User_ID: uuid.NewString(),
		Start:   "07-2024",
	}
	m.db[id] = s

	id2 := uuid.NewString()
	s2 := Sub{
		ID:      id2,
		Service: "service_2",
		Price:   700,
		User_ID: s.User_ID,
		Start:   "07-2024",
	}
	m.db[id2] = s2

	t.Run("payload", func(t *testing.T) {
		subs := testListPayload(t, server.URL)

		compareSubs(t, s, subs[0])
		compareSubs(t, s2, subs[1])

	})
}

func testSumPayload(t *testing.T, server_url string, s Sub) int {

	query := fmt.Sprintf("/subs/sum?start_date=%v&end_date=%v", s.Start, *s.End)
	if s.User_ID != "" {
		query += "&user_id=" + s.User_ID
	}
	if s.Service != "" {
		query += "&service_name=" + s.Service
	}

	resp, err := http.Get(server_url + query)
	if err != nil {
		t.Error(err)
	}

	if resp.StatusCode != 200 {
		t.Errorf("expected status: 200, got: %v", resp.StatusCode)
	}

	var r struct {
		Sum int `json:"sum"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&r); err != nil {
		t.Error(err)
	}
	defer resp.Body.Close()

	return r.Sum
}

func TestSumHandler(t *testing.T) {

	var m MockDB
	m.db = make(map[string]Sub)
	db = &m

	server := httptest.NewServer(newRouter())
	defer server.Close()

	t.Run("malformed", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/subs/sum?user_id=123")
		if err != nil {
			t.Error(err)
		}

		if resp.StatusCode != 400 {
			t.Errorf("expected status: 400, got: %v", resp.StatusCode)
		}
	})

	id := uuid.NewString()
	s := Sub{
		ID:      id,
		Service: "service",
		Price:   400,
		User_ID: uuid.NewString(),
		Start:   "07-2024",
	}
	m.db[id] = s

	id2 := uuid.NewString()
	s2 := Sub{
		ID:      id2,
		Service: "service_2",
		Price:   700,
		User_ID: uuid.NewString(),
		Start:   "10-2024",
	}
	m.db[id2] = s2

	id3 := uuid.NewString()
	s3 := Sub{
		ID:      id3,
		Service: "service_3",
		Price:   200,
		User_ID: s2.User_ID,
		Start:   "10-2024",
		End:     func() *string { s := "10-2024"; return &s }(),
	}
	m.db[id3] = s3

	filter := Sub{
		Start: "07-2024",
		End:   func() *string { s := "09-2024"; return &s }(),
	}

	t.Run("payload", func(t *testing.T) {

		if sum := testSumPayload(t, server.URL, filter); sum != 400*3 {
			t.Errorf("expected sum: 1200, got: %v", sum)
		}

		filter.End = func() *string { s := "10-2024"; return &s }()
		if sum := testSumPayload(t, server.URL, filter); sum != 400*4+700+200 {
			t.Errorf("expected sum: 2500, got: %v", sum)
		}

		filter.User_ID = s2.User_ID
		if sum := testSumPayload(t, server.URL, filter); sum != 700+200 {
			t.Errorf("expected sum: 900, got: %v", sum)
		}

		filter.Service = "service_3"
		if sum := testSumPayload(t, server.URL, filter); sum != 200 {
			t.Errorf("expected sum: 200, got: %v", sum)
		}
	})
}
