package tokenutill

func GenerateToken(id int32, name string) string {
	return name + string(id)
}
