package utils

import "encoding/json"

/*
IsJson verifica se uma string fornecida está em um formato JSON válido.
Ela tenta fazer o unmarshal da string em uma estrutura vazia.
Se falhar, retorna o erro; caso contrário, retorna nil.
*/
func IsJson(s string) error {
	var js struct{}

	// Tenta decodificar a string como JSON.
	if err := json.Unmarshal([]byte(s), &js); err != nil {
		return err
	}

	// Retorna nil se a string for um JSON válido.
	return nil
}
