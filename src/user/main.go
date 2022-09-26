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

func (ugsi *UserGrpcServerImpl) SeedUser(ctx context.Context, in *protoUser.User) (*protoUser.SeedUserResponse, error) {
	log.Printf("Received: %v", in.Name)

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

	created_user := &protoUser.User{Name: in.Name, Balance: in.Balance}
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

	response := &protoUser.SeedUserResponse{Id: id}
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

func (ugsi *UserGrpcServerImpl) GetAmount(ctx context.Context, in *protoUser.GetAmountRequest) (*protoUser.GetAmountResponse, error) {
	tx, err := ugsi.conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("conn.Begin Failed: %v", err)
	}

	id := in.GetId()
	var balance int32
	statementSql := `
	SELECT balance FROM users WHERE id = ($1)
	`
	err = tx.QueryRow(context.Background(), statementSql, id).Scan(&balance)
	if err != nil {
		log.Fatalf("Unable to execute the query. %v", err)
	}
	tx.Commit(context.Background())
	response := &protoUser.GetAmountResponse{Balance: balance}
	return response, nil
}

func (ugsi *UserGrpcServerImpl) UpdateUserBalance(ctx context.Context, in *protoUser.UpdateUserBalanceRequest) (*protoUser.UpdateUserBalanceResponse, error) {
	tx, err := ugsi.conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("conn.Begin Failed: %v", err)
	}

	id := in.GetId()
	changeBalance := in.GetBalance()
	var balance int32
	statementSql := `
	Update users SET balance = balance + ($1) WHERE id = ($2)
	RETURNING balance;
	`
	err = tx.QueryRow(context.Background(), statementSql, changeBalance, id).Scan(&balance)
	if err != nil {
		log.Fatalf("Unable to execute the query. %v", err)
	}
	tx.Commit(context.Background())
	response := &protoUser.UpdateUserBalanceResponse{Balance: balance}
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
	database_url := getenv("POSTGRES_CONN_STRING", "postgres://userdb:userdb@localhost:5432/userdb")
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
