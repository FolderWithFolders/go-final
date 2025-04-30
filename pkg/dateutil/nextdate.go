package dateutil

import (
	"errors"
	"fmt"
	"strconv"
	"strings"
	"time"
)

const DateFormat = "20060102"

func NextDate(now time.Time, date string, rule string) (string, error) {
	if rule == "" {
		return "", errors.New("повторение не указано")
	}

	startDate, err := time.Parse(DateFormat, date)
	if err != nil {
		return "", fmt.Errorf("некорректная дата начала: %v", err)
	}

	parts := strings.Fields(rule)
	if len(parts) == 0 {
		return "", errors.New("неверный формат правила")
	}

	switch parts[0] {
	case "d":
		return handleDailyRule(now, startDate, parts)
	case "y":
		return handleYearlyRule(now, startDate)
	case "w":
		return handleWeeklyRule(now, startDate, parts)
	case "m":
		return handleMonthlyRule(now, startDate, parts)
	default:
		return "", errors.New("неподдерживаемый формат правила")
	}
}

func handleDailyRule(now, startDate time.Time, parts []string) (string, error) {
	if len(parts) != 2 {
		return "", errors.New("неверный формат правила 'd'")
	}

	days, err := strconv.Atoi(parts[1])
	if err != nil || days < 1 || days > 400 {
		return "", errors.New("некорректное количество дней")
	}

	date := startDate
	for {
		date = date.AddDate(0, 0, days)
		if afterNow(date, now) {
			break
		}
	}
	return date.Format(DateFormat), nil
}

// Обработка правила "y"
func handleYearlyRule(now, startDate time.Time) (string, error) {
	date := startDate
	for {
		nextYear := date.Year() + 1
		nextDate := time.Date(nextYear, date.Month(), date.Day(), 0, 0, 0, 0, date.Location())

		// Исправление для 29 февраля
		if nextDate.Day() != date.Day() {
			nextDate = time.Date(nextYear, date.Month()+1, 1, 0, 0, 0, 0, date.Location())
		}

		if afterNow(nextDate, now) {
			return nextDate.Format(DateFormat), nil
		}
		date = nextDate
	}
}

func handleWeeklyRule(now, startDate time.Time, parts []string) (string, error) {
	if len(parts) < 2 {
		return "", errors.New("неверный формат правила 'w'")
	}

	days := make(map[int]bool)
	for _, s := range strings.Split(parts[1], ",") {
		d, err := strconv.Atoi(s)
		if err != nil || d < 1 || d > 7 {
			return "", errors.New("недопустимый день недели")
		}
		days[d] = true
	}

	date := startDate
	for {
		currentDay := int(date.Weekday())
		if currentDay == 0 {
			currentDay = 7
		}
		if days[currentDay] && afterNow(date, now) {
			return date.Format(DateFormat), nil
		}
		date = date.AddDate(0, 0, 1)
	}
}

// Обработка правила "m"
func handleMonthlyRule(now, startDate time.Time, parts []string) (string, error) {
	if len(parts) < 2 {
		return "", errors.New("неверный формат правила 'm'")
	}

	daysStr := strings.Split(parts[1], ",")
	var days []int
	for _, s := range daysStr {
		d, err := strconv.Atoi(s)
		if err != nil || (d < -2 || d == 0 || d > 31) {
			return "", errors.New("недопустимый день месяца")
		}
		days = append(days, d)
	}

	var months []int
	if len(parts) > 2 {
		for _, s := range strings.Split(parts[2], ",") {
			m, err := strconv.Atoi(s)
			if err != nil || m < 1 || m > 12 {
				return "", errors.New("недопустимый месяц")
			}
			months = append(months, m)
		}
	}

	date := startDate
	for {
		year, month, _ := date.Date()
		currentMonth := int(month)

		// Проверяем, подходит ли месяц
		if len(months) > 0 && !contains(months, currentMonth) {
			date = time.Date(year, month+1, 1, 0, 0, 0, 0, date.Location())
			continue
		}

		// Ищем ближайший подходящий день в текущем месяце
		var candidates []time.Time
		for _, d := range days {
			var targetDay time.Time
			if d < 0 {
				lastDay := lastDayOfMonth(year, month)
				adjustedDay := lastDay + d + 1
				if adjustedDay < 1 {
					adjustedDay = 1
				}
				targetDay = time.Date(year, month, adjustedDay, 0, 0, 0, 0, date.Location())
			} else {
				maxDay := lastDayOfMonth(year, month)
				if d > maxDay {
					continue
				}
				targetDay = time.Date(year, month, d, 0, 0, 0, 0, date.Location())
			}

			if (targetDay.After(startDate) || targetDay.Equal(startDate)) && afterNow(targetDay, now) {
				candidates = append(candidates, targetDay)
			}
		}

		// Выбираем минимальную подходящую дату
		if len(candidates) > 0 {
			minDate := candidates[0]
			for _, c := range candidates {
				if c.Before(minDate) {
					minDate = c
				}
			}
			return minDate.Format(DateFormat), nil
		}

		// Переход к следующему месяцу
		date = time.Date(year, month+1, 1, 0, 0, 0, 0, date.Location())
	}
}

func lastDayOfMonth(year int, month time.Month) int {
	return time.Date(year, month+1, 0, 0, 0, 0, 0, time.UTC).Day()
}

func contains(slice []int, item int) bool {
	for _, v := range slice {
		if v == item {
			return true
		}
	}
	return false
}

func afterNow(date, now time.Time) bool {
	date = date.Truncate(24 * time.Hour)
	now = now.Truncate(24 * time.Hour)
	return date.After(now)
}
