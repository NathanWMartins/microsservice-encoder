package services

import (
	"encoding/json"
	"microsservico-encoder/domain"
	"microsservico-encoder/framework/utils"
	"os"
	"sync"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/streadway/amqp"
)

// JobWorkerResult representa o resultado de um trabalho executado pelo worker,
// incluindo o job processado, a mensagem da fila e um possível erro.
type JobWorkerResult struct {
	Job     domain.Job
	Message *amqp.Delivery
	Error   error
}

// Mutex é utilizado para evitar condições de corrida ao acessar
// recursos compartilhados, como inserção de vídeo e job no banco de dados.
var Mutex = &sync.Mutex{}

// JobWorker é responsável por processar mensagens recebidas da fila,
// validar e inserir vídeos e jobs no sistema, e iniciar o processamento do job.
func JobWorker(
	messageChannel chan amqp.Delivery, // canal de mensagens recebidas da fila
	returnChan chan JobWorkerResult, // canal de retorno com o resultado do job
	jobService JobService, // serviço que executa operações com vídeos e jobs
	job domain.Job, // estrutura base do job a ser processado
	workerID int, // identificador do worker
) {

	// Exemplo esperado do corpo da mensagem:
	// {
	//     "resource_id":"id do video da pessoa que enviou para nossa fila",
	//     "file_path": "convite.mp4"
	// }

	for message := range messageChannel {

		// Verifica se o corpo da mensagem é um JSON válido.
		err := utils.IsJson(string(message.Body))
		if err != nil {
			returnChan <- returnJobResult(domain.Job{}, message, err)
			continue
		}

		// Faz o parse da mensagem JSON para o objeto Video.
		// Bloqueia a execução concorrente com Mutex.
		Mutex.Lock()
		err = json.Unmarshal(message.Body, &jobService.VideoService.Video)
		jobService.VideoService.Video.ID = uuid.NewV4().String()
		Mutex.Unlock()

		if err != nil {
			returnChan <- returnJobResult(domain.Job{}, message, err)
			continue
		}

		// Valida o vídeo recebido.
		err = jobService.VideoService.Video.Validate()
		if err != nil {
			returnChan <- returnJobResult(domain.Job{}, message, err)
			continue
		}

		// Insere o vídeo no banco de dados (ou outro meio persistente).
		Mutex.Lock()
		err = jobService.VideoService.InsertVideo()
		Mutex.Unlock()
		if err != nil {
			returnChan <- returnJobResult(domain.Job{}, message, err)
			continue
		}

		// Preenche os dados do job com as informações do vídeo processado.
		job.Video = jobService.VideoService.Video
		job.OutputBucketPath = os.Getenv("outputBucketName") // nome do bucket de saída
		job.ID = uuid.NewV4().String()
		job.Status = "STARTING"
		job.CreatedAt = time.Now()

		// Insere o job no repositório (banco de dados, por exemplo).
		Mutex.Lock()
		_, err = jobService.JobRepository.Insert(&job)
		Mutex.Unlock()

		if err != nil {
			returnChan <- returnJobResult(domain.Job{}, message, err)
			continue
		}

		// Inicia o processamento do job.
		jobService.Job = &job
		err = jobService.Start()
		if err != nil {
			returnChan <- returnJobResult(domain.Job{}, message, err)
			continue
		}

		// Envia o resultado do job processado com sucesso.
		returnChan <- returnJobResult(job, message, nil)
	}
}

// returnJobResult encapsula o resultado da execução de um job,
// retornando uma estrutura com o job, a mensagem e o erro, se houver.
func returnJobResult(job domain.Job, message amqp.Delivery, err error) JobWorkerResult {
	result := JobWorkerResult{
		Job:     job,
		Message: &message,
		Error:   err,
	}
	return result
}
