package services

import (
	"context"
	"io"
	"log"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"cloud.google.com/go/storage"
)

/*
Struct que representa o processo de upload de um vídeo para um bucket.
- Paths: caminhos dos arquivos gerados (ex.: fragmentos).
- VideoPath: caminho base onde os arquivos estão localizados.
- OutputBucket: nome do bucket de destino.
- Errors: lista de caminhos que falharam no upload.
*/
type VideoUpload struct {
	Paths        []string
	VideoPath    string
	OutputBucket string
	Errors       []string
}

/*
Construtor para a struct VideoUpload.
Retorna uma instância vazia da estrutura.
*/
func NewVideoUpload() *VideoUpload {
	return &VideoUpload{}
}

/*
Realiza o upload de um único arquivo (objectPath) para o bucket definido em OutputBucket.
Cria o objeto no bucket com permissão pública de leitura.
*/
func (vu *VideoUpload) UploadObject(objectPath string, client *storage.Client, ctx context.Context) error {
	path := strings.Split(objectPath, os.Getenv("localStoragePath")+"/")

	f, err := os.Open(objectPath)
	if err != nil {
		return err
	}
	defer f.Close()

	wc := client.Bucket(vu.OutputBucket).Object(path[1]).NewWriter(ctx)
	wc.ACL = []storage.ACLRule{{Entity: storage.AllUsers, Role: storage.RoleReader}}

	if _, err = io.Copy(wc, f); err != nil {
		return err
	}

	if err := wc.Close(); err != nil {
		return err
	}

	return nil
}

/*
Carrega todos os caminhos de arquivos contidos dentro de VideoPath e
os armazena no slice Paths, ignorando diretórios.
*/
func (vu *VideoUpload) loadPaths() error {
	err := filepath.Walk(vu.VideoPath, func(path string, info os.FileInfo, err error) error {
		if !info.IsDir() {
			vu.Paths = append(vu.Paths, path)
		}
		return nil
	})

	if err != nil {
		return err
	}
	return nil
}

/*
Controla o processo de upload em paralelo dos arquivos encontrados no diretório.
Cria workers com base no nível de concorrência especificado.
Envia sinal via doneUpload assim que um erro ocorrer ou todos os uploads forem concluídos.
*/
func (vu *VideoUpload) ProcessUpload(concurrency int, doneUpload chan string) error {
	in := make(chan int, runtime.NumCPU())
	returnChannel := make(chan string)

	err := vu.loadPaths()
	if err != nil {
		return err
	}

	uploadClient, ctx, err := getClientUpload()
	if err != nil {
		return err
	}

	for process := 0; process < concurrency; process++ {
		go vu.uploadWorker(in, returnChannel, uploadClient, ctx)
	}

	go func() {
		for x := 0; x < len(vu.Paths); x++ {
			in <- x
		}
	}()

	countDoneWorker := 0
	for r := range returnChannel {
		countDoneWorker++

		if r != "" {
			doneUpload <- r
			break
		}

		if countDoneWorker == len(vu.Paths) {
			close(in)
		}
	}

	return nil
}

/*
Worker responsável por fazer o upload dos arquivos indicados pelo canal `in`.
Se ocorrer erro durante o upload, ele é registrado e enviado via returnChan.
Após terminar, envia uma mensagem final de conclusão.
*/
func (vu *VideoUpload) uploadWorker(in chan int, returnChan chan string, uploadClient *storage.Client, ctx context.Context) {
	for x := range in {
		err := vu.UploadObject(vu.Paths[x], uploadClient, ctx)

		if err != nil {
			vu.Errors = append(vu.Errors, vu.Paths[x])
			log.Printf("error during the upload: %v. Error: %v", vu.Paths[x], err)
			returnChan <- err.Error()
		}

		returnChan <- ""
	}

	returnChan <- "upload completed"
}

/*
Cria e retorna um cliente autenticado para fazer upload de arquivos
utilizando a biblioteca da Google Cloud Storage.
*/
func getClientUpload() (*storage.Client, context.Context, error) {
	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return nil, nil, err
	}
	return client, ctx, nil
}
