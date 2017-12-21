// Copyright 2016 The go-daylight Authors
// This file is part of the go-daylight library.
//
// The go-daylight library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-daylight library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// along with the go-daylight library. If not, see <http://www.gnu.org/licenses/>.

package parser

import (
	"encoding/json"
	"strings"

	"github.com/AplaProject/go-apla/packages/consts"
	"github.com/AplaProject/go-apla/packages/converter"
	"github.com/AplaProject/go-apla/packages/model"
	"github.com/AplaProject/go-apla/packages/utils"

	log "github.com/sirupsen/logrus"
)

func (p *Parser) autoRollback() error {
	logger := p.GetLogger()
	rollbackTx := &model.RollbackTx{}
	txs, err := rollbackTx.GetRollbackTransactions(p.DbTransaction, p.TxHash)
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("getting rollback transactions")
		return utils.ErrInfo(err)
	}
	for _, tx := range txs {
		where := " WHERE id='" + tx["table_id"] + `'`
		if len(tx["data"]) > 0 {
			var jsonMap map[string]string
			err = json.Unmarshal([]byte(tx["data"]), &jsonMap)
			if err != nil {
				logger.WithFields(log.Fields{"type": consts.JSONUnmarshallError, "error": err}).Error("unmarshalling rollback.Data from json")
				return p.ErrInfo(err)
			}
			addSQLUpdate := ""
			for k, v := range jsonMap {
				if converter.InSliceString(k, []string{"hash", "pub", "tx_hash", "public_key_0", "node_public_key"}) && len(v) != 0 {
					addSQLUpdate += k + `=decode('` + string(converter.BinToHex([]byte(v))) + `','HEX'),`
				} else {
					addSQLUpdate += k + `='` + strings.Replace(v, `'`, `''`, -1) + `',`
				}
			}
			addSQLUpdate = addSQLUpdate[0 : len(addSQLUpdate)-1]
			err = model.Update(p.DbTransaction, tx["table_name"], addSQLUpdate, where)
			if err != nil {
				logger.WithFields(log.Fields{"type": consts.JSONUnmarshallError, "error": err, "query": addSQLUpdate}).Error("updating table")
				return p.ErrInfo(err)
			}
		} else {
			err = model.Delete(tx["table_name"], where)
			if err != nil {
				logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("deleting from table")
				return p.ErrInfo(err)
			}
		}
	}
	txForDelete := &model.RollbackTx{TxHash: p.TxHash}
	err = txForDelete.DeleteByHash(p.DbTransaction)
	if err != nil {
		logger.WithFields(log.Fields{"type": consts.DBError, "error": err}).Error("deleting rollback transaction by hash")
		return p.ErrInfo(err)
	}
	return nil
}
