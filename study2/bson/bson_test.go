package bson

import (
	"fmt"
	"testing"
	"time"

	"github.com/gogf/gf/v2/frame/g"
	"go.mongodb.org/mongo-driver/bson"
)

type User struct {
	Uid      any `json:"uid"      bson:"uid"`
	Nickname any `json:"nickname" bson:"nickname"`
	AddTime  any `json:"add_time" bson:"add_time"`
}

func TestOmitempty(t *testing.T) {
	user := User{
		Nickname: "",
		AddTime: bson.M{
			"$gte": time.Now().AddDate(0, 0, -1),
			"$lte": time.Now(),
		},
	}

	r := OmitEmpty(user)
	fmt.Println(r)

	r = OmitEmpty(g.Map{"uid": 0, "nickname": "test"})
	fmt.Println(r)

	t.Fail()
}
