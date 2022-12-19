package main

import(
	"encoding/json"
	"github.com/gorilla/mux"
	"log"
	"net/http"
	"crypto/md5"
	"io"
	"fmt"
	"time"
	"crypto/sha256"
	"encoding/hex"
)

type Service struct {
	Name string `json:"name"`
	ID string `json:"id"`
	Price int `json:"price"`
	ISBN string `json:"isbn"`
	CreatedAt string `json:"createdAt"`
}

type Block struct {
	Position int
	Hash string
	PrevHash string
	Timestamp string
	Transactions []Tx
}

type Tx struct {
	ServiceID string `json:"serviceID"`
	User string `json:"user"`
	CheckoutDate string `json:"checkoutDate"`
	IsGenesis bool `json:"isGenesis"`
}

type Blockchain struct {
	blocks []*Block
}

var blockchain *Blockchain

var TxPool []Tx

var maxPoolSize = 2

func (b *Block) calculateHash() string {
	bytes, _ := json.MarshalIndent(b.Transactions, "", "  ")
	Data := string(bytes) + b.PrevHash + b.Timestamp + string(b.Position)
	hash := sha256.New()
	hash.Write([]byte(Data))
	return hex.EncodeToString(hash.Sum(nil))
}

func CreateBlock(prevBlock *Block, isGenesis bool) *Block {
	if(isGenesis){
		block := &Block{
			Position: prevBlock.Position + 1,
			PrevHash: prevBlock.Hash,
			Timestamp: time.Now().String(),
			Transactions: []Tx{
				{
					ServiceID: "",
					User: "",
					CheckoutDate: "",
					IsGenesis: true,
				},
			},
		}
		block.Hash = block.calculateHash()
		return block
	}
	block := &Block{
		Position: prevBlock.Position + 1,
		PrevHash: prevBlock.Hash,
		Timestamp: time.Now().String(),
		Transactions: TxPool,
	}

	block.Hash = block.calculateHash()
	return block
}



func (b *Blockchain) AddBlock() {
	prevBlock := b.blocks[len(b.blocks)-1]
	block := CreateBlock(prevBlock, false)

	if validBlock(block, prevBlock) {
		b.blocks = append(b.blocks, block)
	}
}

func validBlock(block, prevBlock *Block) bool {
	if prevBlock.Hash != block.PrevHash {
		return false
	}

	if prevBlock.Position+1 != block.Position {
		return false
	}

	if block.calculateHash() != block.Hash {
		return false
	}

	return true
}

func newService(w http.ResponseWriter, r *http.Request) {
	var service Service
	err := json.NewDecoder(r.Body).Decode(&service)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	service.CreatedAt = time.Now().String()
	h := md5.New()
	io.WriteString(h, service.ISBN+service.CreatedAt)
	service.ID = fmt.Sprintf("%x", h.Sum(nil))
	

	resp, err := json.MarshalIndent(service, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)

}

func getBlockchain(w http.ResponseWriter, r *http.Request) {
	resp, err := json.MarshalIndent(blockchain.blocks, "", "  ")
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Write(resp)
}

func writeBlock(w http.ResponseWriter, r *http.Request) {
	var tx Tx
	err := json.NewDecoder(r.Body).Decode(&tx)
	if err != nil {
		http.Error(w, err.Error(), http.StatusBadRequest)
		return
	}
	tx.CheckoutDate = time.Now().String()
	tx.IsGenesis = false
	tx.User = "user1"
	if(len(TxPool) < maxPoolSize){
		TxPool = append(TxPool, tx)
		if(len(TxPool) == maxPoolSize){
			blockchain.AddBlock()
			TxPool = []Tx{}
		}
		w.WriteHeader(http.StatusCreated)
		return
	}else {
		blockchain.AddBlock()
		TxPool = []Tx{}
		w.WriteHeader(http.StatusCreated)
		return
	}
	
}

func GenesisBlock() *Block {
	return CreateBlock(&Block{}, true)
}

func NewBlockchain() *Blockchain {
	return &Blockchain{[]*Block{GenesisBlock()}}
}

func main() {

	blockchain = NewBlockchain()

	r := mux.NewRouter()
	r.HandleFunc("/", getBlockchain).Methods("GET")
	r.HandleFunc("/", writeBlock).Methods("POST")
	r.HandleFunc("/new", newService).Methods("POST")

	go func() {
		for _, block := range blockchain.blocks {
			fmt.Printf("Prev. hash: %x\n", block.PrevHash)
			bytes, _ := json.MarshalIndent(block.Transactions, "", "  ")
			fmt.Printf("Data: %s\n", string(bytes))
			fmt.Printf("Hash: %x\n", block.Hash)
			fmt.Println()
		}
	}()

	log.Println("Listening on port 3000")

	log.Fatal(http.ListenAndServe(":3000", r))
}