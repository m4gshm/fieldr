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
	if r := v.BaseStruct; r != nil {
		m[BaseStructID] = r.ID
	}
	if r := v.BaseStruct; r != nil {
		if r := r.TS; r != nil {
			m[BaseStructTS] = r
		}
	}
	m[Name] = v.Name
	m[Surname] = v.Surname
	m[NoTag] = v.NoTag
	if r := v.Address; r != nil {
		m[Address] = r.AsMap()
	}
	m[FlatCardNum] = v.Flat.CardNum
	m[FlatBank] = v.Flat.Bank
	return m
}

const Flat StructField = "Flat"
