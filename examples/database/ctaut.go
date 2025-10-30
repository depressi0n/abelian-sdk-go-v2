package database

type Metadata struct {
	ID             int64
	RegisteredTxID string
	Version        uint32
	Identifier     string
	Name           string
	Symbol         string
	BaseUnitName   string
	SubUnitName    string
	UnitScale      uint64
	Memo           string

	PlannedTotalAmount      uint64
	IssuerTokens            string
	MintThreshold           uint8
	ReregistrationThreshold uint8
	ExpireHeight            int32

	MintedAmount uint64
	BurnedAmount uint64
}

func InsertCTAUTInstance(metadata *Metadata) (int64, error) {
	// check exist firstly
	exist, err := db.Query(`SELECT id FROM metadata WHERE identifier = ?`, metadata.Identifier)
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

	stmt, err := db.Prepare(`INSERT INTO metadata (registered_tx_id,version,identifier,name,symbol,base_unit_name,sub_unit_name,unit_scale,memo,planned_total_amount,issuer_tokens,mint_threshold,reregistration_threshold,expire_height,minted_amount,burned_amount) 
									VALUES (?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)`)
	if err != nil {
		return -1, err
	}
	result, err := stmt.Exec(
		metadata.RegisteredTxID,
		metadata.Version,
		metadata.Identifier,
		metadata.Name,
		metadata.Symbol,
		metadata.BaseUnitName,
		metadata.SubUnitName,
		metadata.UnitScale,
		metadata.Memo,
		metadata.PlannedTotalAmount,
		metadata.IssuerTokens,
		metadata.MintThreshold,
		metadata.ReregistrationThreshold,
		metadata.ExpireHeight,
		metadata.MintedAmount,
		metadata.BurnedAmount,
	)
	if err != nil {
		return -1, err
	}
	return result.LastInsertId()
}
func LoadCTAUTMetadata(identifier string) (*Metadata, error) {
	rows, err := db.Query(`SELECT id,registered_tx_id,version,identifier,name,symbol,base_unit_name,sub_unit_name,unit_scale,memo,planned_total_amount,issuer_tokens,mint_threshold,reregistration_threshold,expire_height,minted_amount,burned_amount
								 FROM metadata  
								WHERE identifier = ?`, identifier)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	metadata := &Metadata{}
	for rows.Next() {
		var id int64
		var registeredTxID string
		var version uint32
		var identifier string
		var name string
		var symbol string
		var baseUnitName string
		var subUnitName string
		var unitScale uint64
		var memo string
		var plannedTotalAmount uint64
		var issuerTokens string
		var mintThreshold uint8
		var reregistrationThreshold uint8
		var expireHeight int32
		var mintedAmount uint64
		var burnedAmount uint64

		err = rows.Scan(
			&id,
			&registeredTxID,
			&version,
			&identifier,
			&name,
			&symbol,
			&baseUnitName,
			&subUnitName,
			&unitScale,
			&memo,
			&plannedTotalAmount,
			&issuerTokens,
			&mintThreshold,
			&reregistrationThreshold,
			&expireHeight,
			&mintedAmount,
			&burnedAmount,
		)
		if err != nil {
			return nil, err
		}

		metadata.ID = id
		metadata.RegisteredTxID = registeredTxID
		metadata.Version = version
		metadata.Identifier = identifier
		metadata.Name = name
		metadata.Symbol = symbol
		metadata.BaseUnitName = baseUnitName
		metadata.SubUnitName = subUnitName
		metadata.UnitScale = unitScale
		metadata.Memo = memo
		metadata.PlannedTotalAmount = plannedTotalAmount
		metadata.IssuerTokens = issuerTokens
		metadata.MintThreshold = mintThreshold
		metadata.ReregistrationThreshold = reregistrationThreshold
		metadata.ExpireHeight = expireHeight
		metadata.MintedAmount = mintedAmount
		metadata.BurnedAmount = burnedAmount
	}
	return metadata, err
}
func UpdateCTAUTMetadata(metadata *Metadata) error {
	stmt, err := db.Prepare(`UPDATE metadata SET memo = ?, planned_total_amount = ?, issuer_tokens = ?, mint_threshold = ?, reregistration_threshold = ?, expire_height = ?, minted_amount = ?, burned_amount = ? WHERE id = ?`)
	if err != nil {
		return err
	}
	_, err = stmt.Exec(
		metadata.Memo,
		metadata.PlannedTotalAmount,
		metadata.IssuerTokens,
		metadata.MintThreshold,
		metadata.ReregistrationThreshold,
		metadata.ExpireHeight,
		metadata.MintedAmount,
		metadata.BurnedAmount,
		metadata.ID,
	)
	return err
}

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
