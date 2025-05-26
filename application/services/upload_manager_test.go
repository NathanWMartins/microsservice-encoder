package services_test

import (
	"log"
	"microsservico-encoder/application/services"
	"os"
	"testing"

	"github.com/joho/godotenv"
	"github.com/stretchr/testify/require"
)

/*
Função init executada automaticamente antes dos testes.
Carrega as variáveis de ambiente definidas no arquivo .env localizado dois níveis acima.
Se o arquivo não for encontrado, a execução é interrompida com log de erro.
*/
func init() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

/*
Testa todo o fluxo de upload de um vídeo:
- Realiza download do vídeo.
- Fragmenta e codifica.
- Inicializa o upload com concorrência.
- Espera a confirmação de finalização.
- Finaliza o processo do serviço.
Verifica se não há erros em nenhuma das etapas.
*/
func TestVideoServiceUpload(t *testing.T) {

	video, repo := prepare()

	videoService := services.NewVideoService()
	videoService.Video = video
	videoService.VideoRepository = repo

	err := videoService.Download("codeeducationtest")
	require.Nil(t, err)

	err = videoService.Fragment()
	require.Nil(t, err)

	err = videoService.Encode()
	require.Nil(t, err)

	videoUpload := services.NewVideoUpload()
	videoUpload.OutputBucket = "codeeducationtest"
	videoUpload.VideoPath = os.Getenv("localStoragePath") + "/" + video.ID

	doneUpload := make(chan string)
	go videoUpload.ProcessUpload(50, doneUpload)

	result := <-doneUpload
	require.Equal(t, result, "upload completed")

	err = videoService.Finish()
	require.Nil(t, err)
}
