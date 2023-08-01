package user

import (
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/config"
	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
	"github.com/gocraft/dbr/v2"
)

type identitieDB struct {
	session *dbr.Session
	ctx     *config.Context
}

func newIdentitieDB(ctx *config.Context) *identitieDB {
	return &identitieDB{
		session: ctx.DB(),
		ctx:     ctx,
	}
}

func (i *identitieDB) saveOrUpdateTx(m *identitiesModel, tx *dbr.Tx) error {
	_, err := tx.InsertBySql("insert into signal_identities(uid,identity_key,signed_prekey_id,signed_pubkey,signed_signature,registration_id) values(?,?,?,?,?,?) ON DUPLICATE KEY UPDATE identity_key=identity_key,signed_prekey_id=signed_prekey_id,signed_pubkey=signed_pubkey,signed_signature=signed_signature,registration_id=registration_id", m.UID, m.IdentityKey, m.SignedPrekeyID, m.SignedPubkey, m.SignedSignature, m.RegistrationID).Exec()
	return err
}

func (i *identitieDB) deleteWithUID(uid string) error {
	_, err := i.session.DeleteFrom("signal_identities").Where("uid=?", uid).Exec()
	return err
}

func (i *identitieDB) queryWithUID(uid string) (*identitiesModel, error) {
	var model *identitiesModel
	_, err := i.session.Select("*").From("signal_identities").Where("uid=?", uid).Load(&model)
	return model, err
}

type identitiesModel struct {
	UID             string
	RegistrationID  uint32
	IdentityKey     string
	SignedPrekeyID  int
	SignedPubkey    string
	SignedSignature string
	db.BaseModel
}
