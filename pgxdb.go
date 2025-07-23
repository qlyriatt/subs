package subs

import (
	"context"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
)

type PGXDB struct {
	conn *pgx.Conn
}

func NewPGXDB(conn_str string) (*PGXDB, error) {

	conn, err := pgx.Connect(context.Background(), conn_str)
	if err != nil {
		return nil, err
	}

	return &PGXDB{conn: conn}, nil
}

func (db *PGXDB) Create(sub Sub) (string, error) {

	id := uuid.NewString()
	_, err := db.conn.Exec(context.Background(),
		"INSERT INTO subs (sub_id, service_name, price, user_id, start_date, end_date) VALUES ($1, $2, $3, $4, to_date($5, 'MM-YYYY'), to_date($6, 'MM-YYYY'))",
		id, sub.Service, sub.Price, sub.User_ID, sub.Start, sub.End)

	if err != nil {
		return "", err
	}

	return id, nil
}

func (db *PGXDB) Read(id string) (Sub, error) {

	var sub Sub
	err := db.conn.QueryRow(context.Background(),
		"SELECT sub_id, service_name, price, user_id, to_char(start_date, 'MM-YYYY'), to_char(end_date, 'MM-YYYY') FROM subs WHERE sub_id=$1", id).
		Scan(&sub.ID, &sub.Service, &sub.Price, &sub.User_ID, &sub.Start, &sub.End)

	return sub, err
}

func (db *PGXDB) Update(id string, sub Sub) error {

	_, err := db.conn.Exec(context.Background(),
		"UPDATE subs SET service_name=$1, price=$2, user_id=$3, start_date=to_date($4, 'MM-YYYY'), end_date=to_date($5, 'MM-YYYY') WHERE sub_id=$6",
		sub.Service, sub.Price, sub.User_ID, sub.Start, sub.End, id)

	return err
}

func (db *PGXDB) Delete(id string) error {

	_, err := db.conn.Exec(context.Background(),
		"DELETE FROM subs WHERE sub_id=$1", id)

	return err
}

func (db *PGXDB) List() ([]Sub, error) {

	rows, err := db.conn.Query(context.Background(),
		"SELECT sub_id, service_name, price, user_id, to_char(start_date, 'MM-YYYY'), to_char(end_date, 'MM-YYYY') FROM subs")

	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var ss []Sub
	for rows.Next() {
		var sub Sub
		if err := rows.Scan(&sub.ID, &sub.Service, &sub.Price, &sub.User_ID, &sub.Start, &sub.End); err != nil {
			return nil, err
		}
		ss = append(ss, sub)
	}

	return ss, nil
}

func (db *PGXDB) Sum(filter Sub) (int, error) {

	var user_id any
	var service_name any

	if filter.User_ID == "" {
		user_id = nil
	} else {
		user_id = filter.User_ID
	}
	if filter.Service == "" {
		service_name = nil
	} else {
		service_name = filter.Service
	}

	var sum int
	err := db.conn.QueryRow(context.Background(),
		"SELECT sum_in_period(to_date($1, 'MM-YYYY'), to_date($2, 'MM-YYYY'), $3, $4)", filter.Start, filter.End, user_id, service_name).Scan(&sum)
	if err != nil {
		return 0, err
	}

	return sum, nil
}
