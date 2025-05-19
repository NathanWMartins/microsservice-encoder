package services

import (
	"context"
	"io/ioutil"
	"log"
	"microsservico-encoder/application/repositories"
	"microsservico-encoder/domain"
	"os"
	"os/exec"

	"cloud.google.com/go/storage"
)

/*
VideoService é uma estrutura que encapsula a lógica de serviço
relacionada a vídeos, incluindo operações como download, fragmentação,
codificação e limpeza. Ela depende de um repositório de vídeos para persistência.
*/
type VideoService struct {
	Video           *domain.Video
	VideoRepository repositories.VideoRepository
}

/*
NewVideoService cria uma nova instância de VideoService com valores padrão.
*/
func NewVideoService() VideoService {
	return VideoService{}
}

/*
Download baixa o arquivo de vídeo do Google Cloud Storage,
com base no nome do bucket e no caminho do arquivo presente em Video.FilePath,
e o armazena localmente como um arquivo .mp4.
*/
func (v *VideoService) Download(bucketName string) error {

	ctx := context.Background()

	client, err := storage.NewClient(ctx)
	if err != nil {
		return err
	}

	bkt := client.Bucket(bucketName)
	obj := bkt.Object(v.Video.FilePath)

	r, err := obj.NewReader(ctx)
	if err != nil {
		return err
	}
	defer r.Close()

	body, err := ioutil.ReadAll(r)
	if err != nil {
		return err
	}

	f, err := os.Create(os.Getenv("localStoragePath") + "/" + v.Video.ID + ".mp4")
	if err != nil {
		return err
	}

	_, err = f.Write(body)
	if err != nil {
		return err
	}

	defer f.Close()

	log.Printf("video %v has been stored", v.Video.ID)

	return nil
}

/*
Fragment cria uma pasta para armazenar os fragmentos do vídeo e,
em seguida, usa o comando `mp4fragment` para fragmentar o vídeo .mp4
em um arquivo .frag, necessário para a próxima etapa de codificação.
*/
func (v *VideoService) Fragment() error {

	err := os.Mkdir(os.Getenv("localStoragePath")+"/"+v.Video.ID, os.ModePerm)
	if err != nil {
		return err
	}

	source := os.Getenv("localStoragePath") + "/" + v.Video.ID + ".mp4"
	target := os.Getenv("localStoragePath") + "/" + v.Video.ID + ".frag"

	cmd := exec.Command("mp4fragment", source, target)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return err
	}

	printOutput(output)

	return nil
}

/*
Encode utiliza o comando `mp4dash` para codificar o vídeo fragmentado
(.frag) em múltiplos segmentos e manifestos, preparando-o para
streaming adaptativo (DASH).
*/
func (v *VideoService) Encode() error {
	cmdArgs := []string{}
	cmdArgs = append(cmdArgs, os.Getenv("localStoragePath")+"/"+v.Video.ID+".frag")
	cmdArgs = append(cmdArgs, "--use-segment-timeline")
	cmdArgs = append(cmdArgs, "-o")
	cmdArgs = append(cmdArgs, os.Getenv("localStoragePath")+"/"+v.Video.ID)
	cmdArgs = append(cmdArgs, "-f")
	cmdArgs = append(cmdArgs, "--exec-dir")
	cmdArgs = append(cmdArgs, "/opt/bento4/bin/")
	cmd := exec.Command("mp4dash", cmdArgs...)

	output, err := cmd.CombinedOutput()

	if err != nil {
		return err
	}

	printOutput(output)

	return nil
}

/*
Finish remove todos os arquivos temporários gerados durante o processo
(mp4 original, arquivo .frag e pasta de saída), liberando espaço em disco.
*/
func (v *VideoService) Finish() error {

	err := os.Remove(os.Getenv("localStoragePath") + "/" + v.Video.ID + ".mp4")
	if err != nil {
		log.Println("error removing mp4 ", v.Video.ID, ".mp4")
		return err
	}

	err = os.Remove(os.Getenv("localStoragePath") + "/" + v.Video.ID + ".frag")
	if err != nil {
		log.Println("error removing frag ", v.Video.ID, ".frag")
		return err
	}

	err = os.RemoveAll(os.Getenv("localStoragePath") + "/" + v.Video.ID)
	if err != nil {
		log.Println("error removing mp4 ", v.Video.ID, ".mp4")
		return err
	}

	log.Println("files have been removed: ", v.Video.ID)

	return nil

}

/*
InsertVideo insere as informações do vídeo no repositório,
armazenando seus metadados em uma base de dados, por exemplo.
*/
func (v *VideoService) InsertVideo() error {
	_, err := v.VideoRepository.Insert(v.Video)

	if err != nil {
		return err
	}

	return nil
}

/*
printOutput imprime a saída dos comandos executados no terminal,
se houver alguma mensagem ou erro retornado.
*/
func printOutput(out []byte) {
	if len(out) > 0 {
		log.Printf("=====> Output: %s\n", string(out))
	}
}
