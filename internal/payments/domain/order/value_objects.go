package order

// OrderID identifica unicamente um agregado Order.
type OrderID string

func (id OrderID) String() string { return string(id) }
func (id OrderID) IsZero() bool   { return id == "" }
