package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	protoUser "simple-grpc-transcode/proto/user"
	tokenutill "simple-grpc-transcode/src/user/token-utill"

	"github.com/jackc/pgx/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":8081"
)

type UserGrpcServerImpl struct {
	conn *pgx.Conn
	protoUser.UnimplementedUserManagementServer
}

func (ugsi *UserGrpcServerImpl) SeedUser(ctx context.Context, in *protoUser.SeedUserRequest) (*protoUser.SeedUserResponse, error) {
	log.Printf("Received: %v", in.GetName())

	createSql := `
	create table if not exists users(
		id SERIAL PRIMARY KEY,
		name text,
		balance int
	);
	`
	_, err := ugsi.conn.Exec(context.Background(), createSql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Table creation failed: %v\n", err)
		os.Exit(1)
	}

	created_user := &protoUser.SeedUserRequest{Name: in.GetName(), Balance: in.GetBalance()}
	tx, err := ugsi.conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("conn.Begin Failed: %v", err)
	}

	var id int32
	statementSql := `
	insert into users(name, balance) 
	values ($1, $2) RETURNING id;
	`
	err = tx.QueryRow(context.Background(), statementSql, created_user.Name, created_user.Balance).Scan(&id)

	if err != nil {
		log.Fatalf("Unable to execute the query. %v", err)
	}
	tx.Commit(context.Background())

	response := &protoUser.SeedUserResponse{Name: in.Name, Id: id}
	return response, nil
}

func (ugsi *UserGrpcServerImpl) GetUser(ctx context.Context, in *protoUser.GetUserRequest) (*protoUser.GetUserResponse, error) {
	tx, err := ugsi.conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("conn.Begin Failed: %v", err)
	}

	id := in.GetId()
	name := ""
	statementSql := `
	SELECT name FROM users WHERE id = ($1)
	`
	err = tx.QueryRow(context.Background(), statementSql, id).Scan(&name)
	if err != nil {
		log.Fatalf("Unable to execute the query. %v", err)
	}
	tx.Commit(context.Background())
	token := tokenutill.GenerateToken(id, name)
	response := &protoUser.GetUserResponse{Token: token}
	return response, nil
}

var (
	grpcServerImpl *UserGrpcServerImpl            = &UserGrpcServerImpl{}
	_              protoUser.UserManagementServer = grpcServerImpl
)

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func main() {
	// pgsql connection
	database_url := getenv("POSTGRES_CONN_STRING", "postgres://postgres:postgres@localhost:5432/postgres")
	conn, err := pgx.Connect(context.Background(), database_url)
	if err != nil {
		log.Fatalf("Unable to establish connection: %v", err)
	}
	defer conn.Close(context.Background())
	log.Printf("Connection established")

	// grpc server
	listen, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	log.Printf("Grpc server listening on port: %s", port)
	grpcSrv := grpc.NewServer()
	grpcServerImpl.conn = conn
	protoUser.RegisterUserManagementServer(grpcSrv, grpcServerImpl)

	reflection.Register(grpcSrv)
	if err := grpcSrv.Serve(listen); err != nil {
		log.Fatalf("Failed to serve")
	}
}
