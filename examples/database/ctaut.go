package database

type Token struct {
	ID            int64
	AccountID     int64
	Identifier    string
	TxID          string
	Index         uint8
	IsRootToken   bool
	TokenType     uint8
	Version       uint32
	ValueScript   []byte
	CryptoValuePK []byte
	CryptoValueSK []byte
	Value         uint64
	Status        int // 0-immature 1-spendable 2-spent 3-confirmed 4-invalid
}

func InsertToken(
	accountID int64,
	identifier string,
	txID string,
	index uint8,
	isRootToken bool,
	tokenType uint8,
	valueScript []byte,
	cryptoValuePK []byte,
	cryptoValueSK []byte,
	value int64,
	version uint32,
) (int64, error) {
	// check exist firstly
	exist, err := db.Query(`SELECT id FROM ctaut WHERE account_id = ? AND identifier = ? AND tx_id = ? AND output_index = ?`, accountID, identifier, txID, index)
	if err != nil {
		return -1, err
	}
	defer exist.Close()
	if exist.Next() {
		var id int64
		err := exist.Scan(&id)
		if err != nil {
			return -1, err
		}
		return id, err
	}
	stmt, err := db.Prepare(`INSERT INTO ctaut (account_id, identifier, tx_id, output_index, is_root_token, token_type, version, value_script, value_pk, value_sk, value, status) 
									VALUES (?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return -1, err
	}
	result, err := stmt.Exec(
		accountID,
		identifier,
		txID,
		index,
		isRootToken,
		tokenType,
		version,
		valueScript,
		cryptoValuePK,
		cryptoValueSK,
		value,
		0)
	if err != nil {
		return -1, err
	}
	return result.LastInsertId()
}

func LoadCTAUTTokenByAccountID(id int64, identifier string, isRootToken bool) ([]*Token, error) {
	rows, err := db.Query(`SELECT id,account_id,identifier,tx_id,output_index,is_root_token,token_type,version,value_script,value_pk,value_sk,value,status
								 FROM ctaut  
								WHERE account_id = ? AND identifier=? AND is_root_token = ? AND status = 1`,
		id, identifier, isRootToken)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]*Token, 0)
	for rows.Next() {
		var ID int64
		var accountID int64
		var identifier string
		var txID string
		var outputIndex uint8
		var isRootToken bool
		var tokenType uint8
		var version uint32
		var valueScript []byte
		var cryptoValuePK []byte
		var cryptoValueSK []byte
		var value uint64
		var status int

		err = rows.Scan(
			&ID,
			&accountID,
			&identifier,
			&txID,
			&outputIndex,
			&isRootToken,
			&tokenType,
			&version,
			&valueScript,
			&cryptoValuePK,
			&cryptoValueSK,
			&value,
			&status,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, &Token{
			ID:            ID,
			AccountID:     accountID,
			Identifier:    identifier,
			TxID:          txID,
			Index:         outputIndex,
			IsRootToken:   isRootToken,
			TokenType:     tokenType,
			Version:       version,
			ValueScript:   valueScript,
			CryptoValuePK: cryptoValuePK,
			CryptoValueSK: cryptoValueSK,
			Value:         value,
			Status:        status,
		})
	}
	return tokens, err
}

func updateCTAUTTokenStatus(id int64, status int) error {
	stmt, err := db.Prepare("UPDATE ctaut SET status = ? WHERE id = ?")
	if err != nil {
		return err
	}
	_, err = stmt.Exec(
		status,
		id)
	return err
}

func MatureToken(id int64) error {
	return updateCTAUTTokenStatus(id, 1)
}
func SpendToken(id int64) error {
	return updateCTAUTTokenStatus(id, 2)
}

func ConfirmSpentToken(id int64) error {
	return updateCTAUTTokenStatus(id, 3)
}

func InvalidSpentToken(id int64) error {
	return updateCTAUTTokenStatus(id, 4)
}
func LoadTokenByPoint(accountID int64, txID string, index uint8) (*Token, error) {
	rows, err := db.Query(`SELECT id,identifier,is_root_token,token_type,version,value_script,value_pk,value_sk,value,status
								 FROM ctaut  
								WHERE account_id = ? AND tx_id = ? AND output_index = ?`, accountID, txID, index)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	if !rows.Next() {
		return nil, nil
	}

	var ID int64
	//var accountID int64
	var identifier string
	//var txID string
	//var outputIndex uint8
	var isRootToken bool
	var tokenType uint8

	var version uint32
	var valueScript []byte
	var cryptoValuePK []byte
	var cryptoValueSK []byte
	var value uint64
	var status int

	err = rows.Scan(
		&ID,
		//&accountID,
		&identifier,
		//&txID,
		//&outputIndex,
		&isRootToken,
		&tokenType,
		&version,
		&valueScript,
		&cryptoValuePK,
		&cryptoValueSK,
		&value,
		&status,
	)
	if err != nil {
		return nil, err
	}
	return &Token{
		ID:            ID,
		AccountID:     accountID,
		Identifier:    identifier,
		TxID:          txID,
		Index:         index,
		IsRootToken:   isRootToken,
		TokenType:     tokenType,
		Version:       version,
		ValueScript:   valueScript,
		CryptoValuePK: cryptoValuePK,
		CryptoValueSK: cryptoValueSK,
		Value:         value,
		Status:        status,
	}, nil

}

func DisableRootToken(accountID int64, identifier string, txID string) ([]*Token, error) {
	rows, err := db.Query(`SELECT id, tx_id, output_index,is_root_token,token_type,version,value_script,value_pk,value_sk,value,status
       							FROM ctaut  
								WHERE account_id = ? AND identifier = ? AND is_root_token = ? AND tx_id != ?`,
		accountID, identifier, true, txID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	tokens := make([]*Token, 0)
	for rows.Next() {

		var ID int64
		//var accountID int64
		//var identifier string
		var txID2 string
		var outputIndex uint8
		var isRootToken bool
		var tokenType uint8
		var version uint32
		var valueScript []byte
		var cryptoValuePK []byte
		var cryptoValueSK []byte
		var value uint64
		var status int

		err = rows.Scan(
			&ID,
			//&accountID,
			//&identifier,
			&txID2,
			&outputIndex,
			&isRootToken,
			&tokenType,
			&version,
			&valueScript,
			&cryptoValuePK,
			&cryptoValueSK,
			&value,
			&status,
		)
		if err != nil {
			return nil, err
		}
		tokens = append(tokens, &Token{
			ID:            ID,
			AccountID:     accountID,
			Identifier:    identifier,
			TxID:          txID2,
			Index:         outputIndex,
			IsRootToken:   isRootToken,
			TokenType:     tokenType,
			Version:       version,
			ValueScript:   valueScript,
			CryptoValuePK: cryptoValuePK,
			CryptoValueSK: cryptoValueSK,
			Value:         value,
			Status:        status})
	}
	return tokens, err
}
