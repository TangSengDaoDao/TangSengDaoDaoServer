package webhook

// import (
// 	"os"
// 	"testing"

// 	"github.com/stretchr/testify/assert"
// 	"github.com/TangSengDaoDao/TangSengDaoDaoServer/internal/api/group"
// 	"github.com/TangSengDaoDao/TangSengDaoDaoServer/modules/user"
// 	"github.com/TangSengDaoDao/TangSengDaoDaoServerLib/pkg/db"
// )

// func TestGetThirdName(t *testing.T) {
// 	os.Remove("test.db")
// 	s := db.NewSqlite("test.db", "../../../assets/sql")
// 	d := NewDB(s)
// 	udb := user.NewDB(s)
// 	err := udb.Insert(&user.Model{
// 		UID:  "1",
// 		Name: "test1",
// 	})
// 	assert.NoError(t, err)

// 	err = udb.Insert(&user.Model{
// 		UID:  "2",
// 		Name: "test2",
// 	})
// 	assert.NoError(t, err)

// 	fdb := friend.NewDB(s)
// 	err = fdb.Insert(&friend.Model{
// 		UID:    "1",
// 		ToUID:  "2",
// 		Remark: "dddd",
// 	})
// 	assert.NoError(t, err)

// 	err = fdb.Insert(&friend.Model{
// 		UID:    "2",
// 		ToUID:  "1",
// 		Remark: "11dddd",
// 	})
// 	assert.NoError(t, err)

// 	gdb := group.NewDB(s)
// 	gdb.InsertMember(&group.MemberModel{
// 		GroupNo: "g1",
// 		UID:     "1",
// 		Remark:  "g1_name",
// 	})

// 	name, remark, nameInGroup, err := d.GetThirdName("1", "2", "g1")
// 	assert.NoError(t, err)

// 	assert.Equal(t, "test1", name)
// 	assert.Equal(t, "11dddd", remark)
// 	assert.Equal(t, "g1_name", nameInGroup)

// }
