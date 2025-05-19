package repositories

import (
	"fmt"
	"microsservico-encoder/domain"

	"github.com/jinzhu/gorm"
	uuid "github.com/satori/go.uuid"
)

/*
Interface que define os métodos que qualquer repositório de vídeos deve implementar.
Abstrai as operações de persistência de um vídeo.
*/
type VideoRepository interface {
	Insert(video *domain.Video) (*domain.Video, error) // Insere um novo vídeo no banco
	Find(id string) (*domain.Video, error)             // Busca um vídeo por ID
}

// Estrutura concreta que implementa VideoRepository usando o GORM como ORM.
type VideoRepositoryDb struct {
	Db *gorm.DB // Conexão com o banco de dados
}

// Função construtora que retorna uma instância de VideoRepositoryDb
func NewVideoRepository(db *gorm.DB) *VideoRepositoryDb {
	return &VideoRepositoryDb{Db: db}
}

/*
Método que insere um novo vídeo no banco de dados.
Se o ID do vídeo estiver vazio, gera um UUID novo.
Retorna o vídeo inserido ou um erro, se ocorrer.
*/
func (repo VideoRepositoryDb) Insert(video *domain.Video) (*domain.Video, error) {

	if video.ID == "" {
		video.ID = uuid.NewV4().String()
	}

	err := repo.Db.Create(video).Error

	if err != nil {
		return nil, err
	}

	return video, nil
}

/*
Método que busca um vídeo no banco de dados com base no ID.
Usa Preload para carregar os jobs associados ao vídeo.
Retorna um erro se o vídeo não for encontrado.
*/
func (repo VideoRepositoryDb) Find(id string) (*domain.Video, error) {

	var video domain.Video
	repo.Db.Preload("Jobs").First(&video, "id = ?", id)

	if video.ID == "" {
		return nil, fmt.Errorf("video does not exist")
	}

	return &video, nil
}
