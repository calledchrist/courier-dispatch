package order_domain

// OrderId correlates independently assignable stores belonging to one order.
type OrderProps struct {
	OrderId  string
	Total    uint
	Customer Customer
}
