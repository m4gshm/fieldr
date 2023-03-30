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
	if bs := v.BaseStruct; bs != nil {
		m[BaseStructID] = bs.ID
	}
	if bs := v.BaseStruct; bs != nil {
		if ts := bs.TS; ts != nil {
			m[BaseStructTS] = ts
		}
	}
	m[Name] = v.Name
	m[Surname] = v.Surname
	m[NoTag] = v.NoTag
	if a := v.Address; a != nil {
		m[Address] = a.AsMap()
	}
	m[FlatCardNum] = v.Flat.CardNum
	m[FlatBank] = v.Flat.Bank
	return m
}
