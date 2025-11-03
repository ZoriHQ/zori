package classifier

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

type ElementCategory string

const (
	ElementCategoryCTA        ElementCategory = "cta"
	ElementCategoryNavigation ElementCategory = "navigation"
	ElementCategoryContent    ElementCategory = "content"
	ElementCategoryForm       ElementCategory = "form"
	ElementCategoryMedia      ElementCategory = "media"
	ElementCategoryOther      ElementCategory = "other"
)

type Classification struct {
	ElementType     ElementType
	ElementCategory ElementCategory
	IsCTAClick      bool
	LinkDestination *string
	IsExternalLink  bool
	IsDownloadLink  bool
	FormAction      *string
	ImageAlt        *string
	VideoSource     *string
	AriaLabel       *string
}
