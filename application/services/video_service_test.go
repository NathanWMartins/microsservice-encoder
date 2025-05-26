package services_test

import (
	"log"
	"microsservico-encoder/application/repositories"
	"microsservico-encoder/application/services"
	"microsservico-encoder/domain"
	"microsservico-encoder/framework/database"
	"testing"
	"time"

	"github.com/joho/godotenv"
	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

/*
Função init é executada automaticamente ao iniciar o pacote.
Aqui ela carrega as variáveis de ambiente definidas no arquivo .env para uso nos testes.
*/
func init() {
	err := godotenv.Load("../../.env")
	if err != nil {
		log.Fatalf("Error loading .env file")
	}
}

/*
Função auxiliar que prepara o ambiente de teste.
Cria um novo vídeo e instancia um repositório com conexão ao banco de dados de teste.
*/
func prepare() (*domain.Video, repositories.VideoRepositoryDb) {
	db := database.NewDbTest()
	defer db.Close()

	video := domain.NewVideo()
	video.ID = uuid.NewV4().String()
	video.FilePath = "convite.mp4"
	video.CreatedAt = time.Now()

	repo := repositories.VideoRepositoryDb{Db: db}

	return video, repo
}

/*
Função de teste principal que valida o fluxo completo do serviço de vídeo:
download, fragmentação, codificação e finalização.
Usa require.Nil para garantir que nenhum erro ocorra em cada etapa.
*/
func TestVideoServiceDownload(t *testing.T) {
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

	err = videoService.Finish()
	require.Nil(t, err)
}
