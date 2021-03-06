package mysql

import (
	"database/sql"
	"strconv"
	"time"

	"github.com/upframe/shopy"
	"github.com/jmoiron/sqlx"
)

// OrderService ...
type OrderService struct {
	DB *sqlx.DB
}

var ordersMap = map[string]string{
	"ID":                    "o.id",
	"User.ID":               "o.user_id",
	"PayPal":                "o.paypal",
	"Value":                 "o.value",
	"PaymentStatus":         "o.payment_status",
	"FulfillmentStatus":     "o.fulfillment_status",
	"Credits":               "o.credits",
	"Promocode.ID":          "p.id",
	"Promocode.Code":        "p.code",
	"Promocode.Expires":     "p.expires",
	"Promocode.Discount":    "p.discount",
	"Promocode.Percentage":  "p.percentage",
	"Promocode.Deactivated": "p.deactivated",
}

// Get ...
func (s *OrderService) Get(id int) (*shopy.Order, error) {
	orders, err := s.GetsWhere(0, 0, "ID", "ID", strconv.Itoa(id))
	if err != nil {
		return &shopy.Order{}, err
	}

	if len(orders) == 0 {
		return &shopy.Order{}, sql.ErrNoRows
	}

	return orders[0], nil
}

// GetByPayPal ...
func (s *OrderService) GetByPayPal(token string) (*shopy.Order, error) {
	orders, err := s.GetsWhere(0, 0, "ID", "PayPal", token)
	if err != nil {
		return &shopy.Order{}, err
	}

	if len(orders) == 0 {
		return &shopy.Order{}, sql.ErrNoRows
	}

	return orders[0], nil
}

// Gets ...
func (s *OrderService) Gets(first, limit int, order string) ([]*shopy.Order, error) {
	return s.GetsWhere(first, limit, order, "", "")
}

var orderBaseSelectQuery = "SELECT " +
	"o.id AS `order_id`," +
	"o.user_id AS `order_user`," +
	"o.paypal AS `order_paypal`," +
	"o.value AS `order_value`," +
	"o.payment_status AS `order_payment_status`," +
	"o.fulfillment_status AS `order_fulfillment_status`," +
	"o.credits AS `order_credits`," +
	"pc.id AS `promocode_id`," +
	"pc.code AS `promocode_code`," +
	"pc.expires AS `promocode_expires`," +
	"pc.discount AS `promocode_discount`," +
	"pc.percentage AS `promocode_percentage`," +
	"pc.deactivated AS `promocode_deactivated` " +
	"FROM " +
	"orders AS o " +
	"LEFT JOIN " +
	"promocodes AS pc ON o.promocode_id = pc.id"

// GetsWhere ...
func (s *OrderService) GetsWhere(first, limit int, order, where, sth string) ([]*shopy.Order, error) {
	var (
		rows *sql.Rows
		err  error
	)

	orders := []*shopy.Order{}
	order = fieldsToColumns(ordersMap, order)[0]

	if where == "" {
		if limit == 0 {
			rows, err = s.DB.Query(orderBaseSelectQuery+" ORDER BY ?", order)
		} else {
			rows, err = s.DB.Query(orderBaseSelectQuery+" ORDER BY ? LIMIT ? OFFSET ?", order, limit, first)
		}
	} else {
		where = fieldsToColumns(ordersMap, where)[0]

		if limit == 0 {
			rows, err = s.DB.Query(orderBaseSelectQuery+" WHERE "+where+"=? ORDER BY ?", sth, order)
		} else {
			rows, err = s.DB.Query(orderBaseSelectQuery+" WHERE "+where+"=? ORDER BY ? LIMIT ? OFFSET ?", sth, order, limit, first)
		}
	}

	if err != nil {
		return orders, err
	}

	defer rows.Close()

	for rows.Next() {
		order := &shopy.Order{Products: []*shopy.OrderProduct{}, User: &shopy.User{}}

		var (
			id          sql.NullInt64
			code        sql.NullString
			expires     sql.NullString
			discount    sql.NullInt64
			percentage  sql.NullBool
			deactivated sql.NullBool
		)

		err = rows.Scan(
			&order.ID, &order.User.ID, &order.PayPal, &order.Value, &order.PaymentStatus,
			&order.FulfillmentStatus, &order.Credits, &id, &code, &expires, &discount, &percentage, &deactivated)
		if err != nil {
			return orders, err
		}

		if id.Valid {
			order.Promocode = &shopy.Promocode{
				ID:          int(id.Int64),
				Code:        code.String,
				Discount:    int(discount.Int64),
				Percentage:  percentage.Bool,
				Deactivated: deactivated.Bool,
			}

			var t time.Time
			t, err = time.Parse(time.RFC3339, expires.String)
			if err != nil {
				return orders, err
			}

			order.Promocode.Expires = &t
		}

		var rowsps *sql.Rows
		rowsps, err = s.DB.Query("SELECT o.product_id, o.quantity, p.name FROM orders_products AS o INNER JOIN products AS p ON o.product_id = p.id WHERE o.order_id = ?", order.ID)
		defer rowsps.Close()

		for rowsps.Next() {
			prod := &shopy.OrderProduct{}
			rowsps.Scan(&prod.ID, &prod.Quantity, &prod.Name)
			order.Products = append(order.Products, prod)
		}

		orders = append(orders, order)
	}

	err = rows.Err()
	if err != nil {
		return orders, err
	}

	return orders, nil
}

// Total ...
func (s *OrderService) Total() (int, error) {
	return getTableCount(s.DB, "orders")
}

// Create ...
func (s *OrderService) Create(o *shopy.Order) error {
	var (
		res sql.Result
		err error
	)

	if o.Promocode == nil {
		res, err = s.DB.Exec(
			"INSERT INTO orders (`user_id`, `value`, `payment_status`, `fulfillment_status`, `paypal`) VALUES (?, ?, ?, ?, ?)",
			o.User.ID,
			o.Value,
			o.PaymentStatus,
			o.FulfillmentStatus,
			o.PayPal,
		)
	} else {
		res, err = s.DB.Exec(
			"INSERT INTO orders (`user_id`, `promocode_id`, `value`, `payment_status`, `fulfillment_status`, `paypal`) VALUES (?, ?, ?, ?, ?, ?)",
			o.User.ID,
			o.Promocode.ID,
			o.Value,
			o.PaymentStatus,
			o.FulfillmentStatus,
			o.PayPal,
		)
	}

	if err != nil {
		return err
	}

	id, err := res.LastInsertId()
	if err != nil {
		return err
	}

	o.ID = int(id)

	if len(o.Products) == 0 {
		return nil
	}

	query := "INSERT INTO orders_products (order_id, product_id, quantity) VALUES (?, ?, ?)"
	args := []interface{}{o.ID, o.Products[0].ID, o.Products[0].Quantity}

	for i := 1; i < len(o.Products); i++ {
		query += ", (?, ?, ?)"
		args = append(args, o.ID, o.Products[i].ID, o.Products[i].Quantity)
	}

	_, err = s.DB.Exec(query, args...)
	return err
}

var ordersUpdateMap = map[string]string{
	"ID":                "id",
	"User":              "user_id",
	"PayPal":            "paypal",
	"Value":             "value",
	"PaymentStatus":     "payment_status",
	"FulfillmentStatus": "fulfillment_status",
	"Credits":           "credits",
	"Promocode":         "promocode_id",
}

type updateOrder struct {
	ID                int           `db:"id"`
	UserID            int           `db:"user_id"`
	PromocodeID       sql.NullInt64 `db:"promocode_id"`
	PayPal            string        `db:"paypal"`
	PaymentStatus     int16         `db:"payment_status"`
	FulfillmentStatus int16         `db:"fulfillment_status"`
	Value             int           `db:"value"`
	Credits           int           `db:"credits"`
}

// Update ...
func (s *OrderService) Update(o *shopy.Order, fields ...string) error {
	obj := &updateOrder{
		ID:                o.ID,
		UserID:            0,
		PromocodeID:       sql.NullInt64{Valid: false},
		PayPal:            o.PayPal,
		PaymentStatus:     o.PaymentStatus,
		FulfillmentStatus: o.FulfillmentStatus,
		Value:             o.Value,
		Credits:           o.Credits,
	}

	if o.User != nil {
		obj.UserID = o.User.ID
	}

	if o.Promocode != nil {
		obj.PromocodeID.Valid = true
		obj.PromocodeID.Int64 = int64(o.Promocode.ID)
	}

	_, err := s.DB.NamedExec(updateQuery("orders", "id", fieldsToColumns(ordersUpdateMap, fields...)), obj)
	// NOTE: we don't allow the code to update the Products. If there is some
	// problem with an order, it should be cancealed and new one should
	// be created.
	return err
}

// Delete ...
func (s *OrderService) Delete(id int) error {
	o, err := s.Get(id)
	if err != nil {
		return err
	}

	o.PaymentStatus = shopy.OrderCanceled
	o.FulfillmentStatus = shopy.OrderCanceled
	return s.Update(o, "Status")
}
