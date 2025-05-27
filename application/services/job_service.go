package services

import (
	"errors"
	"microsservico-encoder/application/repositories"
	"microsservico-encoder/domain"
	"os"
	"strconv"
)

type JobService struct {
	Job           *domain.Job
	JobRepository repositories.JobRepository
	VideoService  VideoService
}

/*
Start inicia o processamento do Job. Segue as etapas:
1. Atualiza status para "DOWNLOADING" e faz o download do vídeo.
2. Atualiza status para "FRAGMENTING" e fragmenta o vídeo.
3. Atualiza status para "ENCODING" e codifica o vídeo.
4. Realiza o upload e atualiza o status para "UPLOADING".
5. Finaliza o processamento e atualiza status para "COMPLETED".
Se qualquer etapa falhar, o job é marcado como "FAILED".
*/
func (j *JobService) Start() error {

	err := j.changeJobStatus("DOWNLOADING")

	if err != nil {
		return j.failJob(err)
	}

	err = j.VideoService.Download(os.Getenv("inputBucketName"))

	if err != nil {
		return j.failJob(err)
	}

	err = j.changeJobStatus("FRAGMENTING")

	if err != nil {
		return j.failJob(err)
	}

	err = j.VideoService.Fragment()

	if err != nil {
		return j.failJob(err)
	}

	err = j.changeJobStatus("ENCODING")

	if err != nil {
		return j.failJob(err)
	}

	err = j.VideoService.Encode()

	if err != nil {
		return j.failJob(err)
	}

	err = j.performUpload()

	if err != nil {
		return j.failJob(err)
	}

	err = j.changeJobStatus("FINISHING")

	if err != nil {
		return j.failJob(err)
	}

	err = j.VideoService.Finish()

	if err != nil {
		return j.failJob(err)
	}

	err = j.changeJobStatus("COMPLETED")

	if err != nil {
		return j.failJob(err)
	}

	return nil
}

/*
performUpload executa o upload do vídeo fragmentado para o bucket de saída.
Atualiza o status do Job para "UPLOADING" e aguarda o canal `doneUpload` para validar a conclusão.
Se o resultado não for "upload completed", o Job é marcado como "FAILED".
*/
func (j *JobService) performUpload() error {

	err := j.changeJobStatus("UPLOADING")

	if err != nil {
		return j.failJob(err)
	}

	videoUpload := NewVideoUpload()
	videoUpload.OutputBucket = os.Getenv("outputBucketName")
	videoUpload.VideoPath = os.Getenv("localStoragePath") + "/" + j.VideoService.Video.ID
	concurrency, _ := strconv.Atoi(os.Getenv("CONCURRENCY_UPLOAD"))
	doneUpload := make(chan string)

	go videoUpload.ProcessUpload(concurrency, doneUpload)

	var uploadResult string
	uploadResult = <-doneUpload

	if uploadResult != "upload completed" {
		return j.failJob(errors.New(uploadResult))
	}

	return err
}

/*
changeJobStatus atualiza o status do Job no banco de dados para o valor informado.
Se ocorrer erro ao atualizar, o Job é marcado como "FAILED".
*/
func (j *JobService) changeJobStatus(status string) error {
	var err error

	j.Job.Status = status
	j.Job, err = j.JobRepository.Update(j.Job)

	if err != nil {
		return j.failJob(err)
	}

	return nil
}

/*
failJob marca o Job como "FAILED" e registra a mensagem de erro.
A atualização é salva no banco de dados. Retorna o erro original.
*/
func (j *JobService) failJob(error error) error {

	j.Job.Status = "FAILED"
	j.Job.Error = error.Error()

	_, err := j.JobRepository.Update(j.Job)

	if err != nil {
		return err
	}

	return error
}
