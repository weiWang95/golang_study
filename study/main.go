package main

import (
	"fmt"
	"strings"

	"gitlab.shoplazza.site/common/shoplazza-common/xid"
)

func main() {

	// for i := 0; i < 10; i++ {
	// 	id := xid.Get()
	// 	fmt.Println(id, int(id))
	// }
	// return

	BulkSlice([]string{"1", "2", "3", "4", "5"}, 2, func(d []string) error {
		fmt.Println(d)
		return nil
	})

	return

	orgIds := []uint64{331630605160701534, 330281482482376286, 339967532746169950}
	keys := []string{
		"salesperson_on_checkout",
		"sold_out_tip_on_add_cart",
		"commission_on_order_finish",
		"reason_on_cancel_order",
		"reason_on_cash_box",
	}

	var b strings.Builder
	for _, orgID := range orgIds {
		b.WriteString("insert into org_rules(id,organization_id,rule_key,enabled) value")

		for j, key := range keys {
			if j != 0 {
				b.WriteRune(',')
			}

			b.WriteString(fmt.Sprintf("(%d,%d,'%s',1)", xid.GetByKey(orgID), orgID, key))
		}

		b.WriteRune(';')
		b.WriteRune('\n')
		b.WriteRune('\n')
	}

	fmt.Println(b.String())
}

func BulkSlice(data []string, bulkSize int, fn func(d []string) error) error {
	if len(data) == 0 {
		return nil
	}

	fmt.Println(len(data)/bulkSize, len(data)%bulkSize)
	times := len(data) / bulkSize
	if len(data)%bulkSize > 0 {
		times += 1
	}

	for i := 0; i < times; i += 1 {
		begin := i * bulkSize
		end := (i + 1) * bulkSize
		if end > len(data)-1 {
			end = len(data)
		}
		fmt.Printf("i:%d start:%d,end:%d\n", i, begin, end)
		if err := fn(data[begin:end]); err != nil {
			return err
		}
	}

	return nil
}
