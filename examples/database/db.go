package database

import (
	"database/sql"
	"fmt"
	"os"

	_ "github.com/mattn/go-sqlite3"
	"github.com/pqabelian/abelian-sdk-go-v2/examples/common"
)

var db *sql.DB

func init() {
	filename := common.GetDBFileName()

	dbExist := true
	_, err := os.Stat(filename)
	if err != nil && !os.IsNotExist(err) {
		panic(err)
	}
	if os.IsNotExist(err) {
		dbExist = false
	}

	db, err = sql.Open("sqlite3", filename)
	if err != nil {
		panic(err)
	}

	// create tables
	stmt, err := db.Prepare(`CREATE TABLE IF NOT EXISTS account(
    	ID INTEGER PRIMARY KEY,
    	network_id INTEGER,
    	account_privacy_level INTEGER,
    	spend_key_seed TEXT,
    	sn_key_seed TEXT,
    	value_key_seed TEXT,
    	detector_key TEXT)`)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec()
	if err != nil {
		return
	}
	fmt.Println("Table account created!")

	stmt, err = db.Prepare(`CREATE TABLE IF NOT EXISTS coin(
		ID INTEGER PRIMARY KEY,
		account_id INTEGER,
		transaction_version INTEGER,
		transaction_id TEXT,
		output_index INTEGER,
		coin_value INTEGER,
		serial_number TEXT DEFAULT  '',
		block_id TEXT,
		block_height INTEGER,
		ring_id TEXT DEFAULT  '',
		ring_index INTEGER,
		is_coinbase BOOLEAN,
		status INTEGER,
		data BLOB)`)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec()
	if err != nil {
		return
	}
	fmt.Println("Table coin created!")

	stmt, err = db.Prepare(`CREATE TABLE IF NOT EXISTS ring(
		ID INTEGER PRIMARY KEY,
		ring_id TEXT DEFAULT  '',
		ring_version INTEGER,
		ring_height INTEGER,
		ring_block_ids TEXT,
		ring_size INTEGER,
		from_coinbase BOOLEAN)`)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec()
	if err != nil {
		return
	}
	fmt.Println("Table ring created!")

	stmt, err = db.Prepare(`CREATE TABLE IF NOT EXISTS tx (
		ID INTEGER PRIMARY KEY,
		tx_id TEXT,
		unsigned_transaction TEXT,
		sender_account_ids TEXT,
		signed_transaction TEXT,
		status TEXT)`)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec()
	if err != nil {
		return
	}
	fmt.Println("Table tx created!")

	stmt, err = db.Prepare(`CREATE TABLE IF NOT EXISTS metadata (
    	ID INTEGER PRIMARY KEY,
    	registered_tx_id TEXT,
    	
    	version INTEGER,
    	identifier TEXT,
    	name TEXT,
    	symbol TEXT,
    	base_unit_name TEXT,
    	sub_unit_name TEXT,
    	unit_scale INTEGER,
    	memo TEXT,
    	
    	planned_total_amount INTEGER,
    	issuer_tokens TEXT,
    	privacy_type INTEGER,
    	mint_threshold INTEGER,
    	reregistration_threshold INTEGER,
    	expire_height INTEGER,
    	
    	minted_amount INTEGER,
    	burned_amount INTEGER,
    	update_script_versions TEXT)`)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec()
	if err != nil {
		return
	}
	fmt.Println("Table metadata created!")

	stmt, err = db.Prepare(`CREATE TABLE IF NOT EXISTS ctaut (
		ID INTEGER PRIMARY KEY,
		account_id INTEGER,
		identifier TEXT,
		tx_id TEXT,
		output_index INTEGER,
		is_root_token BOOLEAN,
		token_type INTEGER,
		version INTEGER,
		value_script BLOB,
		value_pk BLOB,
		value_sk BLOB,
		value INTEGER,
		status INTEGER)`)
	if err != nil {
		panic(err)
	}
	_, err = stmt.Exec()
	if err != nil {
		return
	}
	fmt.Println("Table ctaut created!")

	if !dbExist {
		initBuiltInAccount()
	}
}
