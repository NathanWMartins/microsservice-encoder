package domain

import (
	"time"

	"github.com/asaskevich/govalidator"
	uuid "github.com/satori/go.uuid"
)

// init configura o govalidator para exigir todos os campos marcados como obrigatórios por padrão
func init() {
	govalidator.SetFieldsRequiredByDefault(true)
}

/*
Job representa um processo de encoding de vídeo
Cada job está associado a um vídeo e contém informações sobre status, erro e datas de criação e atualização
O campo VideoID é uma foreign key para a tabela de vídeos
O campo Video é um ponteiro para a struct Video, permitindo acesso ao vídeo associado
Os campos são anotados com tags do GORM e validações do govalidator
*/

type Job struct {
	ID               string    `json:"job_id" valid:"uuid" gorm:"type:uuid;primary_key"`     // Identificador único do job
	OutputBucketPath string    `json:"output_bucket_path" valid:"notnull"`                   // Caminho de saída do arquivo processado
	Status           string    `json:"status" valid:"notnull"`                               // Status atual do job (ex: pending, completed)
	Video            *Video    `json:"video" valid:"-"`                                      // Referência ao vídeo associado
	VideoID          string    `json:"-" valid:"-" gorm:"column:video_id;type:uuid;notnull"` // Chave estrangeira para o vídeo
	Error            string    `valid:"-"`                                                   // Mensagem de erro, se houver
	CreatedAt        time.Time `json:"created_at" valid:"-"`                                 // Data de criação
	UpdatedAt        time.Time `json:"updated_at" valid:"-"`                                 // Data da última atualização
}

/*
NewJob cria uma nova instância de Job, executa a preparação e validação
Retorna um ponteiro para o Job criado ou erro caso a validação falhe
*/
func NewJob(output string, status string, video *Video) (*Job, error) {

	job := Job{
		OutputBucketPath: output,
		Status:           status,
		Video:            video,
	}

	job.prepare()

	err := job.Validate()

	if err != nil {
		return nil, err
	}

	return &job, nil
}

/*
prepare define os valores padrão de um job (ID, datas)
Essa função é privada (letra minúscula) e só pode ser chamada dentro do mesmo pacote
*/
func (job *Job) prepare() {
	job.ID = uuid.NewV4().String()
	job.CreatedAt = time.Now()
	job.UpdatedAt = time.Now()
}

/*
Validate executa a validação do job usando o govalidator
Retorna erro caso a validação falhe
*/
func (job *Job) Validate() error {
	_, err := govalidator.ValidateStruct(job)

	if err != nil {
		return err
	}

	return nil
}
