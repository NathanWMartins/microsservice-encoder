package services

import (
	"encoding/json"
	"log"
	"microsservico-encoder/application/repositories"
	"microsservico-encoder/domain"
	"microsservico-encoder/framework/queue"
	"os"
	"strconv"

	"github.com/jinzhu/gorm"
	"github.com/streadway/amqp"
)

/*
JobManager é responsável por orquestrar os workers, coordenando a
recepção de mensagens, execução de jobs e envio de notificações.
*/
type JobManager struct {
	Db               *gorm.DB             // Conexão com o banco de dados
	Domain           domain.Job           // Estrutura do job que será processado
	MessageChannel   chan amqp.Delivery   // Canal com mensagens recebidas da fila
	JobReturnChannel chan JobWorkerResult // Canal de retorno dos resultados dos workers
	RabbitMQ         *queue.RabbitMQ      // Cliente para comunicação com RabbitMQ
}

/*
JobNotificationError representa a estrutura da notificação de erro
que será enviada para outra fila via RabbitMQ.
*/
type JobNotificationError struct {
	Message string `json:"message"`
	Error   string `json:"error"`
}

/*
NewJobManager cria e retorna uma nova instância de JobManager
com todos os canais e conexões necessárias para operação.
*/
func NewJobManager(db *gorm.DB, rabbitMQ *queue.RabbitMQ, jobReturnChannel chan JobWorkerResult, messageChannel chan amqp.Delivery) *JobManager {
	return &JobManager{
		Db:               db,
		Domain:           domain.Job{},
		MessageChannel:   messageChannel,
		JobReturnChannel: jobReturnChannel,
		RabbitMQ:         rabbitMQ,
	}
}

/*
Start inicializa os workers de acordo com a variável de ambiente CONCURRENCY_WORKERS.
Cada worker processa mensagens da fila e envia o resultado via canal.
Ao final do processamento, as mensagens são confirmadas ou rejeitadas.
*/
func (j *JobManager) Start(ch *amqp.Channel) {

	videoService := NewVideoService()
	videoService.VideoRepository = repositories.VideoRepositoryDb{Db: j.Db}

	jobService := JobService{
		JobRepository: repositories.JobRepositoryDb{Db: j.Db},
		VideoService:  videoService,
	}

	concurrency, err := strconv.Atoi(os.Getenv("CONCURRENCY_WORKERS"))

	if err != nil {
		log.Fatalf("error loading var: CONCURRENCY_WORKERS.")
	}

	// Inicializa os workers concorrentes com base no valor de CONCURRENCY_WORKERS.
	for qtdProcesses := 0; qtdProcesses < concurrency; qtdProcesses++ {
		go JobWorker(j.MessageChannel, j.JobReturnChannel, jobService, j.Domain, qtdProcesses)
	}

	// Processa os resultados recebidos dos workers.
	for jobResult := range j.JobReturnChannel {
		if jobResult.Error != nil {
			err = j.checkParseErrors(jobResult)
		} else {
			err = j.notifySuccess(jobResult, ch)
		}

		if err != nil {
			jobResult.Message.Reject(false)
		}
	}
}

/*
notifySuccess envia uma notificação de sucesso contendo o job serializado em JSON.
Em seguida, confirma a mensagem na fila com `Ack`.
*/
func (j *JobManager) notifySuccess(jobResult JobWorkerResult, ch *amqp.Channel) error {

	Mutex.Lock()
	jobJson, err := json.Marshal(jobResult.Job)
	Mutex.Unlock()

	if err != nil {
		return err
	}

	err = j.notify(jobJson)

	if err != nil {
		return err
	}

	err = jobResult.Message.Ack(false)

	if err != nil {
		return err
	}

	return nil
}

/*
checkParseErrors trata mensagens com erro, imprimindo logs,
criando uma estrutura de erro e enviando uma notificação para outra fila.
*/
func (j *JobManager) checkParseErrors(jobResult JobWorkerResult) error {
	if jobResult.Job.ID != "" {
		log.Printf("MessageID: %v. Error during the job: %v with video: %v. Error: %v",
			jobResult.Message.DeliveryTag, jobResult.Job.ID, jobResult.Job.Video.ID, jobResult.Error.Error())
	} else {
		log.Printf("MessageID: %v. Error parsing message: %v", jobResult.Message.DeliveryTag, jobResult.Error)
	}

	errorMsg := JobNotificationError{
		Message: string(jobResult.Message.Body),
		Error:   jobResult.Error.Error(),
	}

	jobJson, err := json.Marshal(errorMsg)

	err = j.notify(jobJson)

	if err != nil {
		return err
	}

	err = jobResult.Message.Reject(false)

	if err != nil {
		return err
	}

	return nil
}

/*
notify envia mensagens para uma exchange do RabbitMQ com base
nas variáveis de ambiente RABBITMQ_NOTIFICATION_EX e RABBITMQ_NOTIFICATION_ROUTING_KEY.
*/
func (j *JobManager) notify(jobJson []byte) error {

	err := j.RabbitMQ.Notify(
		string(jobJson),
		"application/json",
		os.Getenv("RABBITMQ_NOTIFICATION_EX"),
		os.Getenv("RABBITMQ_NOTIFICATION_ROUTING_KEY"),
	)

	if err != nil {
		return err
	}

	return nil
}
