package order_domain

type VendorProps struct {
	VendorId    uint
	Title       string
	AddressText string
	City        string
	Location    Location
	Items       []Item
}
