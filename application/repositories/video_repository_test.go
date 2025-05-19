package repositories_test

import (
	"microsservico-encoder/application/repositories"
	"microsservico-encoder/domain"
	"microsservico-encoder/framework/database"
	"testing"
	"time"

	uuid "github.com/satori/go.uuid"
	"github.com/stretchr/testify/require"
)

/*
TestVideoRepositoryDbInsert testa o método Insert do repositório de vídeos
Verifica se o vídeo é inserido corretamente e se pode ser recuperado pelo método Find
*/
func TestVideoRepositoryDbInsert(t *testing.T) {
	db := database.NewDbTest() // Cria instância do banco de dados em memória para testes
	defer db.Close()           // Garante o fechamento da conexão após o teste

	video := domain.NewVideo()       // Cria uma nova instância de vídeo
	video.ID = uuid.NewV4().String() // Define um ID único para o vídeo
	video.FilePath = "path"          // Define o caminho do arquivo do vídeo
	video.CreatedAt = time.Now()     // Define a data de criação do vídeo

	repo := repositories.VideoRepositoryDb{Db: db} // Cria o repositório de vídeo com a conexão de teste
	repo.Insert(video)                             // Insere o vídeo no banco de dados

	v, err := repo.Find(video.ID) // Recupera o vídeo usando o ID inserido

	require.NotEmpty(t, v.ID)        // Verifica se o vídeo recuperado possui ID
	require.Nil(t, err)              // Verifica se não houve erro ao buscar o vídeo
	require.Equal(t, v.ID, video.ID) // Compara o ID do vídeo inserido com o recuperado
}
