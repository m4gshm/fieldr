package asmap

type StructField string

const (
	BaseStructID StructField = "ID"
	BaseStructTS StructField = "TS"
	Name         StructField = "Name"
	Surname      StructField = "Surname"
	NoTag        StructField = "NoTag"
	Address      StructField = "Address"
	FlatCardNum  StructField = "CardNum"
	FlatBank     StructField = "Bank"
)

func (v *Struct[n]) AsMap() map[StructField]interface{} {
	if v == nil {
		return nil
	}
	m := map[StructField]interface{}{}
	if v_bs := v.BaseStruct; v_bs != nil {
		m[BaseStructID] = v_bs.ID
	}
	if v_bs := v.BaseStruct; v_bs != nil {
		if v__ts := v_bs.TS; v__ts != nil {
			m[BaseStructTS] = v__ts
		}
	}
	m[Name] = v.Name
	m[Surname] = v.Surname
	m[NoTag] = v.NoTag
	if v_a := v.Address; v_a != nil {
		m[Address] = v_a.AsMap()
	}
	m[FlatCardNum] = v.Flat.CardNum
	m[FlatBank] = v.Flat.Bank
	return m
}
