package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	protoTransaction "simple-grpc-transcode/proto/transaction"
	protoUser "simple-grpc-transcode/proto/user"
	tokenutill "simple-grpc-transcode/src/user/token-utill"
	"time"

	"github.com/jackc/pgx/v4"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/reflection"
)

const (
	port = ":8082"
)

type TransactionGrpcServerImpl struct {
	conn *pgx.Conn
	protoTransaction.UnimplementedTransactionManagementServer
}

var (
	grpcServerImpl *TransactionGrpcServerImpl                   = &TransactionGrpcServerImpl{}
	_              protoTransaction.TransactionManagementServer = grpcServerImpl
)

func UpdateUserAndGetCurrentBalance(id int32, amount int32) int32 {
	address := getenv("USER_SERVICE", "localhost:8081")
	conn, err := grpc.Dial(address, grpc.WithTransportCredentials(insecure.NewCredentials()))

	if err != nil {
		log.Fatalf("did not connect: %v", err)
	}
	defer conn.Close()
	c := protoUser.NewUserManagementClient(conn)

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()

	r, err := c.UpdateUserBalance(ctx, &protoUser.UpdateUserBalanceRequest{Id: id, Balance: amount})
	if err != nil {
		log.Fatalf("Could not get user balance %v", err)
	}

	return r.Balance
}

func CreateTransaction(tgsi *TransactionGrpcServerImpl, in *protoTransaction.UpdateTxRequest, tx_status string, current_balance int32) {
	createSql := `
	create table if not exists transaction_records(
		user_id int,
		amount int,
		debited_or_credited text,
		current_balance int
	);
	`
	_, err := tgsi.conn.Exec(context.Background(), createSql)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Table creation failed: %v\n", err)
		os.Exit(1)
	}

	created_transaction := &protoTransaction.UpdateTxRequest{Id: in.Id, Amount: in.Amount}
	tx, err := tgsi.conn.Begin(context.Background())
	if err != nil {
		log.Fatalf("conn.Begin Failed: %v", err)
	}

	statementSql := `
	insert into transaction_records(user_id, amount, debited_or_credited, current_balance) 
	values ($1, $2, $3, $4);
	`
	_, err = tx.Exec(context.Background(), statementSql, created_transaction.Id, created_transaction.Amount, tx_status, current_balance)

	if err != nil {
		log.Fatalf("Unable to execute the query. %v", err)
	}
	tx.Commit(context.Background())
}

func (tgsi *TransactionGrpcServerImpl) UpTransaction(ctx context.Context, in *protoTransaction.UpdateTxRequest) (*protoTransaction.UpdateTxResponse, error) {
	log.Printf("Received Id: %v", in.Id)
	current_balance := UpdateUserAndGetCurrentBalance(in.Id, in.Amount)
	CreateTransaction(tgsi, in, "credited", current_balance)
	response := &protoTransaction.UpdateTxResponse{Balance: current_balance}
	return response, nil
}

func (tgsi *TransactionGrpcServerImpl) DownTransaction(ctx context.Context, in *protoTransaction.UpdateTxRequest) (*protoTransaction.UpdateTxResponse, error) {
	log.Printf("Received Id: %v", in.Id)
	current_balance := UpdateUserAndGetCurrentBalance(in.Id, in.Amount*-1)
	// pass negative value of amount
	CreateTransaction(tgsi, in, "debited", current_balance)
	response := &protoTransaction.UpdateTxResponse{Balance: current_balance}
	return response, nil
}

func getenv(key, fallback string) string {
	value := os.Getenv(key)
	if len(value) == 0 {
		return fallback
	}
	return value
}

func main() {
	fmt.Printf("%s\n", tokenutill.GenerateJWK())
	// pgsql connection
	database_url := getenv("POSTGRES_CONN_STRING", "postgres://transactiondb:transactiondb@localhost:5432/transactiondb")
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
	protoTransaction.RegisterTransactionManagementServer(grpcSrv, grpcServerImpl)

	reflection.Register(grpcSrv)
	if err := grpcSrv.Serve(listen); err != nil {
		log.Fatalf("Failed to serve")
	}
}
