package ui

import (
	"fyne.io/fyne/v2"
)

// folderAt returns the folder uid containing point, or "" for none.
func folderAt(point fyne.Position, rects map[string]rect) string {
	for uid, bounds := range rects {
		if point.X >= bounds.x && point.X <= bounds.x+bounds.width &&
			point.Y >= bounds.y && point.Y <= bounds.y+bounds.height {
			return uid
		}
	}
	return ""
}

// dropConnection finishes a drag: the connection moves into the folder under the
// pointer, or to the root when it was dropped outside every folder row.
func (s *screen) dropConnection(uid string, point fyne.Position) {
	profile, known := s.index.profiles[uid]
	if !known {
		return
	}
	driver := fyne.CurrentApp().Driver()
	rects := make(map[string]rect, len(s.rows))
	for rowUID, row := range s.rows {
		if !isFolderUID(rowUID) {
			continue
		}
		position := driver.AbsolutePositionForObject(row)
		size := row.Size()
		rects[rowUID] = rect{x: position.X, y: position.Y, width: size.Width, height: size.Height}
	}
	target := folderAt(point, rects)
	folder := ""
	if target != "" {
		folder = s.index.folderPath(target)
	}
	if profile.Folder == folder {
		return
	}
	s.moveConnection(profile, folder)
}
