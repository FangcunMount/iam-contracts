package handler

import (
	"time"

	"github.com/gin-gonic/gin"

	guarddomain "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/guardianship"
	guardport "github.com/fangcun-mount/iam-contracts/internal/apiserver/domain/guardianship/port"
	requestdto "github.com/fangcun-mount/iam-contracts/internal/apiserver/interface/restful/request"
	responsedto "github.com/fangcun-mount/iam-contracts/internal/apiserver/interface/restful/response"
)

// GuardianshipHandler 监护关系 REST 处理器
type GuardianshipHandler struct {
	*BaseHandler
	manager guardport.GuardianshipManager
	query   guardport.GuardianshipQueryer
}

// NewGuardianshipHandler 创建监护处理器
func NewGuardianshipHandler(
	manager guardport.GuardianshipManager,
	query guardport.GuardianshipQueryer,
) *GuardianshipHandler {
	return &GuardianshipHandler{
		BaseHandler: NewBaseHandler(),
		manager:     manager,
		query:       query,
	}
}

// Grant 授予监护关系
func (h *GuardianshipHandler) Grant(c *gin.Context) {
	var req requestdto.GuardianGrantRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	userID, err := parseUserID(req.UserID)
	if err != nil {
		h.Error(c, err)
		return
	}

	childID, err := parseChildID(req.ChildID)
	if err != nil {
		h.Error(c, err)
		return
	}

	relation, err := parseRelation(req.Relation)
	if err != nil {
		h.Error(c, err)
		return
	}

	if err := h.manager.AddGuardian(c.Request.Context(), childID, userID, relation); err != nil {
		h.Error(c, err)
		return
	}

	guardianship, err := h.query.FindByUserIDAndChildID(c.Request.Context(), userID, childID)
	if err != nil {
		h.Error(c, err)
		return
	}

	h.Success(c, newGuardianshipResponse(guardianship))
}

// Revoke 撤销监护关系
func (h *GuardianshipHandler) Revoke(c *gin.Context) {
	var req requestdto.GuardianRevokeRequest
	if err := h.BindJSON(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	userID, err := parseUserID(req.UserID)
	if err != nil {
		h.Error(c, err)
		return
	}

	childID, err := parseChildID(req.ChildID)
	if err != nil {
		h.Error(c, err)
		return
	}

	if err := h.manager.RemoveGuardian(c.Request.Context(), childID, userID); err != nil {
		h.Error(c, err)
		return
	}

	guardianship, err := h.query.FindByUserIDAndChildID(c.Request.Context(), userID, childID)
	if err != nil {
		h.Error(c, err)
		return
	}

	var revokedAt time.Time
	if guardianship != nil && guardianship.RevokedAt != nil {
		revokedAt = *guardianship.RevokedAt
	} else {
		revokedAt = time.Now()
	}

	h.Success(c, gin.H{
		"id":        guardianship.ID,
		"revokedAt": revokedAt,
	})
}

// List 查询监护关系
func (h *GuardianshipHandler) List(c *gin.Context) {
	var req requestdto.GuardianshipListQuery
	if err := h.BindQuery(c, &req); err != nil {
		h.Error(c, err)
		return
	}

	var guardianships []*guarddomain.Guardianship
	var err error

	switch {
	case req.UserID != "" && req.ChildID != "":
		userID, uerr := parseUserID(req.UserID)
		if uerr != nil {
			h.Error(c, uerr)
			return
		}
		childID, cerr := parseChildID(req.ChildID)
		if cerr != nil {
			h.Error(c, cerr)
			return
		}
		g, qerr := h.query.FindByUserIDAndChildID(c.Request.Context(), userID, childID)
		if qerr != nil {
			h.Error(c, qerr)
			return
		}
		if g != nil {
			guardianships = []*guarddomain.Guardianship{g}
		} else {
			guardianships = []*guarddomain.Guardianship{}
		}
	case req.UserID != "":
		userID, uerr := parseUserID(req.UserID)
		if uerr != nil {
			h.Error(c, uerr)
			return
		}
		guardianships, err = h.query.FindListByUserID(c.Request.Context(), userID)
	case req.ChildID != "":
		childID, cerr := parseChildID(req.ChildID)
		if cerr != nil {
			h.Error(c, cerr)
			return
		}
		guardianships, err = h.query.FindListByChildID(c.Request.Context(), childID)
	default:
		guardianships = []*guarddomain.Guardianship{}
	}

	if err != nil {
		h.Error(c, err)
		return
	}

	filtered := filterGuardianships(guardianships, req.Active)
	total := len(filtered)
	items := make([]responsedto.GuardianshipResponse, 0, total)
	for _, g := range filtered {
		if g == nil {
			continue
		}
		items = append(items, newGuardianshipResponse(g))
	}

	sliced := sliceGuardianships(items, req.Offset, req.Limit)

	h.Success(c, responsedto.GuardianshipPageResponse{
		Total:  total,
		Limit:  req.Limit,
		Offset: req.Offset,
		Items:  sliced,
	})
}

func filterGuardianships(items []*guarddomain.Guardianship, active *bool) []*guarddomain.Guardianship {
	if active == nil {
		return items
	}

	res := make([]*guarddomain.Guardianship, 0, len(items))
	for _, g := range items {
		if g == nil {
			continue
		}
		if *active && !g.IsActive() {
			continue
		}
		if !*active && g.IsActive() {
			continue
		}
		res = append(res, g)
	}
	return res
}

func sliceGuardianships(items []responsedto.GuardianshipResponse, offset, limit int) []responsedto.GuardianshipResponse {
	if limit <= 0 {
		limit = 20
	}
	if offset < 0 {
		offset = 0
	}
	if offset >= len(items) {
		return []responsedto.GuardianshipResponse{}
	}
	end := offset + limit
	if end > len(items) {
		end = len(items)
	}
	return items[offset:end]
}
