package data

type ProductRepo struct {
	data *Data
}

func NewProductRepo(data *Data) *ProductRepo {
	return &ProductRepo{data: data}
}
