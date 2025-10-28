package classifier

// ElementType represents the specific type of HTML element
type ElementType string

const (
	ElementTypeButton      ElementType = "button"
	ElementTypeLink        ElementType = "link"
	ElementTypeInput       ElementType = "input"
	ElementTypeTextElement ElementType = "text_element"
	ElementTypeImage       ElementType = "image"
	ElementTypeVideo       ElementType = "video"
	ElementTypeNavigation  ElementType = "navigation"
	ElementTypeFormControl ElementType = "form_control"
	ElementTypeOther       ElementType = "other"
)

// ElementCategory represents higher-level categorization
type ElementCategory string

const (
	ElementCategoryCTA        ElementCategory = "cta"
	ElementCategoryNavigation ElementCategory = "navigation"
	ElementCategoryContent    ElementCategory = "content"
	ElementCategoryForm       ElementCategory = "form"
	ElementCategoryMedia      ElementCategory = "media"
	ElementCategoryOther      ElementCategory = "other"
)

// Classification represents the computed classification of a click element
type Classification struct {
	ElementType      ElementType
	ElementCategory  ElementCategory
	IsCTAClick       bool
	LinkDestination  *string
	IsExternalLink   bool
	IsDownloadLink   bool
	FormAction       *string
	ImageAlt         *string
	VideoSource      *string
	AriaLabel        *string
}
