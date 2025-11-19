package main

import (
	"context"
	"database/sql"
	"encoding/json"
	"log"
	"net/http"
	"time"

	_ "github.com/mattn/go-sqlite3"
)

type AwesomeAPIResponse struct {
	USDBRL struct {
		Bid string `json:"bid"`
	} `json:"USDBRL"`
}

type ClientResponse struct {
	Bid string `json:"bid"`
}

func main() {
	db, err := sql.Open("sqlite3", "./cotacao.db")
	if err != nil {
		log.Fatal(err)
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS cotacoes (id INTEGER PRIMARY KEY AUTOINCREMENT, valor TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);")
	if err != nil {
		log.Fatal(err)
	}

	http.HandleFunc("/cotacao", cotacaoHandler)

	log.Println("Servidor iniciado na porta :8080. Endpoint: /cotacao")
	if err := http.ListenAndServe(":8080", nil); err != nil {
		log.Fatal("Erro ao iniciar servidor: ", err)
	}
}

func cotacaoHandler(w http.ResponseWriter, r *http.Request) {
	log.Println("Requisição /cotacao recebida.")

	ctxAPI, cancelAPI := context.WithTimeout(r.Context(), 200*time.Millisecond)
	defer cancelAPI()

	cotacao, err := buscarCotacao(ctxAPI)
	if err != nil {
		logarErro(w, "Erro ao buscar cotação da API", err, http.StatusInternalServerError)
		return
	}

	ctxDB, cancelDB := context.WithTimeout(r.Context(), 10*time.Millisecond)
	defer cancelDB()

	if err := persistirCotacao(ctxDB, cotacao.USDBRL.Bid); err != nil {
		logarErro(w, "Erro ao persistir cotação no DB", err, http.StatusInternalServerError)
	}

	response := ClientResponse{Bid: cotacao.USDBRL.Bid}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)

	log.Println("Resposta enviada com sucesso.")
}

func buscarCotacao(ctx context.Context) (*AwesomeAPIResponse, error) {
	req, err := http.NewRequestWithContext(ctx, "GET", "https://economia.awesomeapi.com.br/json/last/USD-BRL", nil)
	if err != nil {
		return nil, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		select {
		case <-ctx.Done():
			log.Printf("ERRO - Timeout 200ms: Chamada à API externa excedeu o tempo limite. Erro: %v", ctx.Err())
			return nil, ctx.Err()
		default:
			return nil, err
		}
	}
	defer resp.Body.Close()

	var result AwesomeAPIResponse
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, err
	}

	log.Printf("Cotação API recebida: %s", result.USDBRL.Bid)
	return &result, nil
}

func persistirCotacao(ctx context.Context, bid string) error {
	db, err := sql.Open("sqlite3", "./cotacao.db")
	if err != nil {
		return err
	}
	defer db.Close()

	_, err = db.Exec("CREATE TABLE IF NOT EXISTS cotacoes (id INTEGER PRIMARY KEY AUTOINCREMENT, valor TEXT, timestamp DATETIME DEFAULT CURRENT_TIMESTAMP);")
	if err != nil {
		return err
	}

	stmt, err := db.PrepareContext(ctx, "INSERT INTO cotacoes(valor) VALUES(?)")
	if err != nil {
		return err
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, bid)

	if err != nil {
		select {
		case <-ctx.Done():
			log.Printf("ERRO - Timeout 10ms: Persistência no DB excedeu o tempo limite. Erro: %v", ctx.Err())
			return ctx.Err()
		default:
			return err
		}
	}

	log.Printf("Cotação persistida no DB: %s", bid)
	return nil
}

func logarErro(w http.ResponseWriter, msg string, err error, status int) {
	log.Printf("%s: %v", msg, err)
	http.Error(w, msg, status)
}
