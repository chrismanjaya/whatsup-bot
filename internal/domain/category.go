package domain

import "fmt"

type Category string

const (
	CategoryFood          Category = "food"
	CategoryGroceries     Category = "groceries"
	CategoryHousehold     Category = "household supplies"
	CategoryMedicine      Category = "medicine"
	CategoryHealth        Category = "health"
	CategoryElectronics   Category = "electronics"
	CategoryFashion       Category = "fashion"
	CategoryHobby         Category = "hobby"
	CategoryEntertainment Category = "entertainment"
	CategoryTransport     Category = "transport"
	CategoryUtilities     Category = "utilities"
	CategoryEducation     Category = "education"
	CategoryHousing       Category = "housing"
	CategoryTax           Category = "tax"
	CategoryInsurance     Category = "insurance"
	CategoryDebt          Category = "debt"
	CategorySalary        Category = "salary"
	CategoryInvestment    Category = "investment"
	CategoryGift          Category = "gift"
	CategoryTransfer      Category = "transfer"
	CategoryOther         Category = "other"
)

// AllCategories is the single source of truth for valid categories.
// Add or remove a category here only — Valid(), the Gemini schema's
// enum, and the prompt's category list all derive from this slice.
var AllCategories = []Category{
	CategoryFood, CategoryGroceries, CategoryHousehold, CategoryMedicine, CategoryHealth,
	CategoryElectronics, CategoryFashion, CategoryHobby, CategoryEntertainment, CategoryTransport,
	CategoryUtilities, CategoryEducation, CategoryHousing, CategoryTax, CategoryInsurance,
	CategoryDebt, CategorySalary, CategoryInvestment, CategoryGift, CategoryTransfer, CategoryOther,
}

func (c Category) String() string { return string(c) }

func (c Category) Valid() bool {
	for _, valid := range AllCategories {
		if c == valid {
			return true
		}
	}
	return false
}

func ParseCategory(s string) (Category, error) {
	c := Category(s)
	if !c.Valid() {
		return "", fmt.Errorf("invalid category: %q", s)
	}
	return c, nil
}
