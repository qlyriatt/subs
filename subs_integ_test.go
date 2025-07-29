package subs

import (
	"context"
	"fmt"
	"log"
	"os"
	"testing"
	"time"

	"github.com/golang-migrate/migrate/v4"
	_ "github.com/golang-migrate/migrate/v4/database/postgres"
	_ "github.com/golang-migrate/migrate/v4/source/file"
	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/testcontainers/testcontainers-go"
)

func setupInteg(t *testing.T) {

	req := testcontainers.ContainerRequest{
		Image: "postgres:17",
		Env: map[string]string{
			"POSTGRES_USER":     "testuser",
			"POSTGRES_PASSWORD": "testpass",
			"POSTGRES_DB":       "testdb",
		},
		ExposedPorts: []string{"5432/tcp"},
	}

	ctx := context.Background()

	container, err := testcontainers.GenericContainer(ctx, testcontainers.GenericContainerRequest{
		ContainerRequest: req,
		Started:          true,
	})
	if err != nil {
		t.Fatal(err)
	}

	host, err := container.Host(ctx)
	if err != nil {
		t.Fatal(err)
	}

	port, err := container.MappedPort(ctx, "5432")
	if err != nil {
		t.Fatal(err)
	}

	// wait for db startup
	if ret := func() int {
		var ret int
		for range 5 {

			time.Sleep(time.Second)
			ret, _, err = container.Exec(ctx, []string{"sh", "-c", "pg_isready -U testuser -d testdb"})
			if err != nil {
				t.Fatal(err)
			}
			if ret == 0 {
				break
			}
		}

		return ret
	}(); ret != 0 {
		t.Fatal("container connection error")
	}

	str := fmt.Sprintf("postgres://testuser:testpass@%s:%s/testdb?sslmode=disable", host, port.Port())

	mig, err := migrate.New("file://", str)
	if err != nil {
		t.Fatal(err)
	}

	if err := mig.Up(); err != nil {
		t.Fatal(err)
	}

	conn, err := pgx.Connect(ctx, str)
	if err != nil {
		t.Fatal(err)
	}

	file, err := os.OpenFile("integ.log", os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		t.Fatal(err)
	}
	SetLogger(log.New(file, "", log.LstdFlags))

	db = &PGXDB{conn: conn}
	go Start(db)
	time.Sleep(time.Millisecond * 500)
}

func TestInteg(t *testing.T) {

	setupInteg(t)

	server_url := "http://localhost:8080"

	s := Sub{
		Service: "service",
		Price:   400,
		User_ID: uuid.NewString(),
		Start:   "07-2024",
		End:     func() *string { s := "09-2024"; return &s }(),
	}
	var id string

	t.Run("create", func(t *testing.T) {
		id = testCreatePayload(t, server_url, s)
	})

	t.Run("read", func(t *testing.T) {
		compareSubs(t, s, testReadPayload(t, server_url, id))
	})

	s2 := Sub{
		Service: "service_2",
		Price:   700,
		User_ID: s.User_ID,
		Start:   "10-2024",
	}

	t.Run("update", func(t *testing.T) {
		testUpdatePayload(t, server_url, id, s2)
		compareSubs(t, s2, testReadPayload(t, server_url, id))
	})

	t.Run("delete", func(t *testing.T) {
		testDeletePayload(t, server_url, id)
	})

	t.Run("list", func(t *testing.T) {

		// list empty db
		testListPayload(t, server_url)

		count := 5
		var ids []string
		for range count {
			ids = append(ids, testCreatePayload(t, server_url, s))
		}

		subs := testListPayload(t, server_url)
		if len(subs) != count {
			t.Errorf("expected list count: %v, got %v", count, len(subs))
		}

		for _, id := range ids {

			found := false
			for _, sub := range subs {
				if sub.ID == id {
					found = true
					compareSubs(t, sub, s)
				}
			}
			if !found {
				t.Errorf("not found sub with id: %v", id)
			}
		}
	})

	t.Run("sum", func(t *testing.T) {

		filter := Sub{
			Start: "07-2024",
			End:   func() *string { s := "09-2024"; return &s }(),
		}

		// 5 entries of s.Price from 07 to 09 of 2024 inclusively
		expected := s.Price * 5 * 3
		if sum := testSumPayload(t, server_url, filter); sum != expected {
			t.Errorf("expected sum: %v, got %v", expected, sum)
		}

		filter.User_ID = s.User_ID
		if sum := testSumPayload(t, server_url, filter); sum != expected {
			t.Errorf("expected sum: %v, got %v", expected, sum)
		}

		filter.Service = s.Service
		if sum := testSumPayload(t, server_url, filter); sum != expected {
			t.Errorf("expected sum: %v, got %v", expected, sum)
		}

		filter.Service = "123"
		if sum := testSumPayload(t, server_url, filter); sum != 0 {
			t.Errorf("expected sum: %v, got %v", 0, sum)
		}

		filter.Service = ""
		filter.Start = "09-2024"
		expected = s.Price * 5 * 1
		if sum := testSumPayload(t, server_url, filter); sum != expected {
			t.Errorf("expected sum: %v, got %v", expected, sum)
		}
	})
}
