package main

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"
)

type ServerResponse struct {
	Bid string `json:"bid"`
}

const (
	serverURL  = "http://localhost:8080/cotacao"
	timeout    = 300 * time.Millisecond
	outputFile = "cotacao.txt"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	log.Printf("Tentando requisitar cotação em %s com timeout de %v...", serverURL, timeout)

	req, err := http.NewRequestWithContext(ctx, "GET", serverURL, nil)
	if err != nil {
		log.Fatalf("Erro ao criar requisição: %v", err)
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			log.Fatalf("ERRO - Timeout 300ms: Requisição ao servidor excedeu o tempo limite: %v", ctx.Err())
		}
		log.Fatalf("Erro ao realizar requisição HTTP: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		bodyBytes, _ := io.ReadAll(resp.Body)
		log.Fatalf("Erro no servidor. Status Code: %d, Resposta: %s", resp.StatusCode, string(bodyBytes))
	}

	var cotacao ServerResponse
	err = json.NewDecoder(resp.Body).Decode(&cotacao)
	if err != nil {
		log.Fatalf("Erro ao decodificar JSON: %v", err)
	}

	log.Printf("Cotação recebida: USD-BRL = %s", cotacao.Bid)

	conteudo := fmt.Sprintf("Dólar: %s\n", cotacao.Bid)

	err = os.WriteFile(outputFile, []byte(conteudo), 0644)
	if err != nil {
		log.Fatalf("Erro ao salvar arquivo '%s': %v", outputFile, err)
	}

	log.Printf("Cotação salva com sucesso no arquivo '%s'.", outputFile)
}
