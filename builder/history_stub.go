package main

import deb ".."

type HistoryStub struct {
	hist []deb.SourcePackageRef
}

func (h *HistoryStub) Append(p deb.SourcePackageRef) {
	h.hist = append([]deb.SourcePackageRef{p}, h.hist...)
}
func (h *HistoryStub) Get() []deb.SourcePackageRef {
	return h.hist
}
func (h *HistoryStub) RemoveFront(p deb.SourcePackageRef) {
	oldHist := h.hist
	h.hist = nil
	for _, oldP := range oldHist {
		if oldP == p {
			continue
		}
		h.hist = append(h.hist, oldP)
	}
}
