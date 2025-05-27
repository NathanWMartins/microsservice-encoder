package main

import (
	"log"
	"microsservico-encoder/application/services"
	"microsservico-encoder/framework/database"
	"microsservico-encoder/framework/queue"
	"os"
	"strconv"

	"github.com/joho/godotenv"
	"github.com/streadway/amqp"
)

// db representa a instância de configuração do banco de dados.
var db database.Database

/*
init é executada automaticamente antes da função main.
Ela carrega variáveis de ambiente e configura o struct de banco de dados.
*/
func init() {
	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file")
	}

	autoMigrateDb, err := strconv.ParseBool(os.Getenv("AUTO_MIGRATE_DB"))
	if err != nil {
		log.Fatalf("Error parsing boolean env var")
	}

	debug, err := strconv.ParseBool(os.Getenv("DEBUG"))
	if err != nil {
		log.Fatalf("Error parsing boolean env var")
	}

	// Configurações do banco de dados extraídas do .env
	db.AutoMigrateDb = autoMigrateDb
	db.Debug = debug
	db.DsnTest = os.Getenv("DSN_TEST")
	db.Dsn = os.Getenv("DSN")
	db.DbTypeTest = os.Getenv("DB_TYPE_TEST")
	db.DbType = os.Getenv("DB_TYPE")
	db.Env = os.Getenv("ENV")
}

/*
main é o ponto de entrada da aplicação.
Ela estabelece conexões com o banco e o RabbitMQ, inicializa os canais de mensagens,
instancia o JobManager e inicia o processamento.
*/
func main() {

	// Canais de comunicação para mensagens da fila e retorno dos jobs
	messageChannel := make(chan amqp.Delivery)
	jobReturnChannel := make(chan services.JobWorkerResult)

	// Conecta ao banco de dados
	dbConnection, err := db.Connect()

	if err != nil {
		log.Fatalf("error connecting to DB")
	}

	defer dbConnection.Close()

	// Inicializa e conecta ao RabbitMQ
	rabbitMQ := queue.NewRabbitMQ()
	ch := rabbitMQ.Connect()
	defer ch.Close()

	// Inicia o consumo de mensagens da fila
	rabbitMQ.Consume(messageChannel)

	// Instancia o JobManager e inicia o processamento dos jobs
	jobManager := services.NewJobManager(dbConnection, rabbitMQ, jobReturnChannel, messageChannel)
	jobManager.Start(ch)
}
