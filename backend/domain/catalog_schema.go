package domain

import "strings"

// CatalogPartSchema describes one reference's content (see PartContent)
// as a JSON Schema (draft 2020-12), built from the same rules Validate
// applies: key shape, label length, the known icons, colours and cycle
// types, country and calling codes. The back office's editor checks,
// completes and documents with it as the super admin types; Validate
// still decides (uniqueness across entries, removed keys). Descriptions
// are in lang ("fr" or "en"). Nil for an unknown reference.
func CatalogPartSchema(name, lang string) map[string]any {
	d := func(fr, en string) string {
		if lang == "en" {
			return en
		}
		return fr
	}
	key := map[string]any{
		"type": "string", "pattern": catalogKey.String(), "maxLength": 60,
		"description": d("Clé stable, enregistrée sur les fermes, annonces et avis : minuscules et chiffres, séparés par - ou _. Ne change plus une fois publiée.",
			"Stable key, stored on farms, listings and reviews: lowercase letters and digits, joined by - or _. Never changes once published."),
	}
	text := map[string]any{"type": "string", "minLength": 1, "maxLength": maxCatalogLabel, "pattern": `\S`}
	label := map[string]any{
		"type": "object", "required": []string{"fr", "en"}, "additionalProperties": false,
		"description": d("Le nom, en français et en anglais.", "The name, in French and English."),
		"properties":  map[string]any{"fr": text, "en": text},
	}
	active := map[string]any{
		"type":        "boolean",
		"description": d("false masque l’entrée des nouveaux choix ; ce qui l’utilise déjà reste valable. On désactive, on ne supprime pas.", "false hides the entry from new choices; what already uses it stays valid. Deactivate, never remove."),
	}
	enum := func(values []string, fr, en string) map[string]any {
		return map[string]any{"type": "string", "enum": values, "description": d(fr+" : "+strings.Join(values, ", "), en+": "+strings.Join(values, ", "))}
	}
	object := func(required []string, props map[string]any, fr, en string) map[string]any {
		return map[string]any{"type": "object", "required": required, "additionalProperties": false, "properties": props, "description": d(fr, en)}
	}
	list := func(item map[string]any, minItems int, fr, en string) map[string]any {
		return map[string]any{"type": "array", "items": item, "minItems": minItems, "description": d(fr, en)}
	}
	term := object([]string{"key", "label", "active"}, map[string]any{"key": key, "label": label, "active": active},
		"Une entrée de la liste.", "One entry of the list.")

	subcategory := object([]string{"key", "label", "cycleType", "icon", "unit", "active"}, map[string]any{
		"key": key, "label": label, "active": active,
		"cycleType": enum(CycleTypes, "Le cycle dans lequel ce type est élevé", "The cycle this kind is raised in"),
		"icon":      enum(AnimalIcons, "L’illustration affichée", "The drawing shown"),
		"unit": object([]string{"key", "label"}, map[string]any{
			"key":   map[string]any{"type": "string", "minLength": 1, "maxLength": 30, "description": d("Clé de l’unité, enregistrée sur les annonces.", "The unit's key, stored on listings.")},
			"label": label,
		}, "Comment s’appelle un animal sur une annonce (« poussin »).", "What one animal is called on a listing (“chick”)."),
	}, "Un type d’animal vendu. Sa clé est unique dans tout le catalogue.", "A kind of animal sold. Its key is unique across the catalog.")

	category := object([]string{"key", "label", "icon", "color", "active", "subcategories"}, map[string]any{
		"key": key, "label": label, "active": active,
		"icon":          enum(AnimalIcons, "L’illustration affichée", "The drawing shown"),
		"color":         enum(CategoryColors, "La couleur de la catégorie", "The category's colour"),
		"subcategories": list(subcategory, 1, "Les types vendus dans cette catégorie, dans l’ordre d’affichage.", "The kinds sold in this category, in display order."),
	}, "Une catégorie, telle que les acheteurs parcourent le marché.", "A category, the way buyers browse the marketplace.")

	city := object([]string{"key", "label", "active"}, map[string]any{"key": key, "label": label, "active": active},
		"Une ville que les fermes peuvent choisir. Sa clé est unique dans le pays.", "A town farms can pick. Its key is unique within the country.")
	region := object([]string{"key", "label", "active"}, map[string]any{
		"key": key, "label": label, "active": active,
		"cities": list(city, 0, "Les villes principales de la région.", "The region's main towns."),
	}, "Une région de premier niveau.", "A first-level region.")
	country := object([]string{"code", "callingCode", "active"}, map[string]any{
		"code":        map[string]any{"type": "string", "pattern": "^[A-Z]{2}$", "description": d("Code ISO 3166-1 à deux lettres (CM).", "Two-letter ISO 3166-1 code (CM).")},
		"callingCode": map[string]any{"type": "string", "pattern": callingCode.String(), "description": d("Indicatif téléphonique, sans + (237).", "Calling code, without + (237).")},
		"active":      active,
		"regions":     list(region, 0, "Sans régions, les fermes saisissent la leur librement.", "Without regions, farms type theirs freely."),
	}, "Un pays où les fermes peuvent se trouver.", "A country farms can be in.")

	root := func(title, field string, items map[string]any) map[string]any {
		return map[string]any{
			"$schema": "https://json-schema.org/draft/2020-12/schema",
			"title":   title,
			"type":    "object", "additionalProperties": false,
			"required":   []string{field},
			"properties": map[string]any{field: items},
		}
	}
	switch name {
	case PartAnimals:
		return root(d("Animaux", "Animals"), "categories", list(category, 1, "Les catégories d’animaux, dans l’ordre d’affichage.", "The animal categories, in display order."))
	case PartPlaces:
		return root(d("Lieux", "Places"), "countries", list(country, 1, "Les pays ; "+DefaultCountry+" doit rester actif.", "The countries; "+DefaultCountry+" must stay active."))
	case PartForumCategories:
		return root(d("Catégories du forum", "Forum categories"), "entries", list(term, 0, "Où un sujet du forum est classé.", "Where a forum topic is filed."))
	case PartReviewTags:
		return root(d("Points forts des avis", "Review highlights"), "entries", list(term, 0, "Ce qu’un acheteur peut souligner dans un avis.", "What a buyer can point out in a review."))
	case PartReportReasons:
		return root(d("Motifs de signalement", "Report reasons"), "entries", list(term, 0, "Pourquoi on signale une annonce, un profil ou un message.", "Why a listing, profile or post is reported."))
	case PartFarmReviewChecks:
		return root(d("Vérification des fermes", "Farm checks"), "entries", list(term, 0, "Les points vérifiés avant d’approuver une ferme.", "What is checked before approving a farm."))
	}
	return nil
}
