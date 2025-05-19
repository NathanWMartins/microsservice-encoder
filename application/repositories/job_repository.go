package repositories

import (
	"fmt"
	"microsservico-encoder/domain"

	"github.com/jinzhu/gorm"
)

// JobRepository define a interface com os métodos para manipulação de Job no banco
type JobRepository interface {
	Insert(job *domain.Job) (*domain.Job, error) // Insere um novo Job e retorna o Job inserido ou erro
	Find(id string) (*domain.Job, error)         // Busca um Job pelo ID, retorna o Job ou erro
	Update(job *domain.Job) (*domain.Job, error) // Atualiza um Job existente e retorna o Job atualizado ou erro
}

// JobRepositoryDb é a implementação da interface JobRepository usando GORM e uma conexão ao banco
type JobRepositoryDb struct {
	Db *gorm.DB // Conexão com o banco de dados via GORM
}

// Insert adiciona um novo registro de Job no banco
func (repo JobRepositoryDb) Insert(job *domain.Job) (*domain.Job, error) {
	err := repo.Db.Create(job).Error // Cria o registro no banco, verifica erro

	if err != nil {
		return nil, err // Retorna erro se falhar
	}

	return job, nil // Retorna o Job inserido
}

// Find busca um Job pelo seu ID no banco, carregando também a referência ao Video (Preload)
func (repo JobRepositoryDb) Find(id string) (*domain.Job, error) {
	var job domain.Job
	repo.Db.Preload("Video").First(&job, "id = ?", id) // Busca o primeiro registro com o ID informado e faz preload do vídeo associado

	if job.ID == "" {
		return nil, fmt.Errorf("job does not exist") // Retorna erro se não encontrar o Job
	}

	return &job, nil // Retorna o Job encontrado
}

// Update atualiza o registro do Job no banco
func (repo JobRepositoryDb) Update(job *domain.Job) (*domain.Job, error) {
	err := repo.Db.Save(&job).Error // Salva as alterações no banco e verifica erro

	if err != nil {
		return nil, err // Retorna erro caso ocorra falha na atualização
	}

	return job, nil // Retorna o Job atualizado
}
