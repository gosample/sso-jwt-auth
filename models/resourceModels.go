package models

import (
	"errors"

	"github.com/asofdate/sso-jwt-auth/utils/logger"
	"github.com/asofdate/sso-jwt-auth/utils/validator"
	"github.com/astaxie/beego/logs"
	"github.com/hzwy23/dbobj"
)

type ResourceModel struct {
	Mtheme ThemeResourceModel
}

type resNodeData struct {
	ResId   string `json:"resId"`
	ResName string `json:"resName"`
	ResUpId string `json:"resUpId"`
}

type ResData struct {
	Res_id        string `json:"res_id"`
	Res_name      string `json:"res_name"`
	Res_attr      string `json:"res_attr"`
	Res_attr_desc string `json:"res_attr_desc"`
	Res_up_id     string `json:"res_up_id"`
	Res_type      string `json:"res_type"`
	Res_type_desc string `json:"res_type_desc"`
	Sys_flag      string `json:"sys_flag"`
	Inner_flag    string `json:"inner_flag"`
	Service_cd    string `json:"service_cd"`
}

// 查询所有的资源信息
func (this *ResourceModel) Get() ([]ResData, error) {
	rows, err := dbobj.Query(sys_rdbms_071)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	var rst []ResData
	err = dbobj.Scan(rows, &rst)
	return rst, err
}

func (this *ResourceModel) GetInnerFlag(resId string) (string, error) {
	innerFlag := "true"
	err := dbobj.QueryForObject(sys_rdbms_079, dbobj.PackArgs(resId), &innerFlag)
	return innerFlag, err
}

func (this *ResourceModel) GetServiceCd(resId string) (string, error) {
	serviceCd := ""
	err := dbobj.QueryForObject(sys_rdbms_048, dbobj.PackArgs(resId), &serviceCd)
	return serviceCd, err
}

func (this *ResourceModel) GetChildren(res_id string) ([]ResData, error) {
	rst, err := this.Get()
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	var ret []ResData
	this.dfs(rst, res_id, &ret)
	return ret, nil
}

// 所有指定资源的详细信息
func (this *ResourceModel) Query(res_id string) ([]ResData, error) {
	rows, err := dbobj.Query(sys_rdbms_089, res_id)
	if err != nil {
		logger.Error(err)
		return nil, err
	}
	var rst []ResData
	err = dbobj.Scan(rows, &rst)
	return rst, err
}

// 新增菜单资源
func (this *ResourceModel) Post(data ResData) (string, error) {
	// 如果所属系统非空，表示是内部菜单
	innnerFlag := "false"
	if len(data.Service_cd) == 0 {
		innnerFlag = "true"
	}

	// 1 表示叶子
	// 0 表示结点
	res_attr := "1"
	if data.Res_type == "0" || data.Res_type == "4" {
		res_attr = "0"
	}

	// 如果是首页子系统菜单，设置上级编码为-1
	if data.Res_type == "0" {
		data.Res_up_id = "-1"
	}

	if !validator.IsWord(data.Res_id) {
		logger.Error("资源编码必须由1,30位字母或数字组成")
		return "error_resource_res_id", errors.New("error_resource_res_id")
	}

	if validator.IsEmpty(data.Res_name) {
		logger.Error("菜单名称不能为空")
		return "error_resource_desc_empty", errors.New("error_resource_desc_empty")
	}

	if validator.IsEmpty(data.Res_type) {
		logger.Error("菜单类别不能为空")
		return "error_resource_type", errors.New("error_resource_type")
	}

	if validator.IsEmpty(data.Res_up_id) {
		logger.Error("菜单上级编码不能为空")
		return "error_resource_up_id", errors.New("error_resource_up_id")
	}

	// add sys_resource_info
	_, err := dbobj.Exec(sys_rdbms_072,
		data.Res_id, data.Res_name, res_attr, data.Res_up_id,
		data.Res_type, innnerFlag, data.Service_cd)

	if err != nil {
		logger.Error(err)
		return "error_resource_add", err
	}
	return "success", nil
}

// 删除指定的资源
func (this *ResourceModel) Delete(res_id string) (string, error) {
	var rst []ResData

	all, err := this.Get()
	if err != nil {
		logger.Error(err)
		return "error_resource_query", err
	}

	this.dfs(all, res_id, &rst)

	// add res_id
	for _, val := range all {
		if val.Res_id == res_id {
			rst = append(rst, val)
			break
		}
	}

	tx, err := dbobj.Begin()
	if err != nil {
		logger.Error(err)
		return "error_resource_begin", err
	}

	for _, val := range rst {

		if val.Sys_flag == "0" {
			tx.Rollback()
			return "error_resource_forbid_system_resource", errors.New("error_resource_forbid_system_resource")
		}

		_, err = tx.Exec(sys_rdbms_075, val.Res_id)
		if err != nil {
			logger.Error(err)
			tx.Rollback()
			return "error_resource_role_relation", err
		}

		_, err = tx.Exec(sys_rdbms_076, val.Res_id)
		if err != nil {
			logger.Error(err)
			tx.Rollback()
			return "error_resource_theme_relation", err
		}

		_, err = tx.Exec(sys_rdbms_077, val.Res_id)
		if err != nil {
			logger.Error(err)
			tx.Rollback()
			return "error_resource_delete", err
		}
	}
	return "error_resource_commit", tx.Commit()
}

func (this *ResourceModel) Update(arg ResData) (string, error) {

	if validator.IsEmpty(arg.Res_name) {
		return "error_resource_desc_empty", errors.New("error_resource_desc_empty")
	}

	if arg.Res_id == arg.Res_up_id {
		return "error_resource_update_same", errors.New("error_resource_update_same")
	}

	//获取当前菜单所有子菜单列表
	childList, err := this.GetChildren(arg.Res_id)
	if err != nil {
		logs.Error(err)
		return "error_resource_update", errors.New("error_resource_update")
	}

	for _, val := range childList {
		if val.Res_id == arg.Res_up_id {
			return "error_resource_update", errors.New("error_resource_update")
		}
	}

	_, err = dbobj.Exec(sys_rdbms_005,
		arg.Res_name,
		arg.Res_up_id,
		arg.Service_cd,
		arg.Res_id)

	if err != nil {
		logger.Error(err)
		return "error_resource_update", err
	}
	return "success", nil
}

func (this *ResourceModel) GetNodes(resId string) ([]resNodeData, error) {
	var rst []resNodeData
	err := dbobj.QueryForSlice(sys_rdbms_046, &rst)
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	childList, err := this.GetChildren(resId)
	if err != nil {
		logs.Error(err)
		return nil, err
	}
	mp := make(map[string]string)
	for _, val := range childList {
		mp[val.Res_id] = ""
	}
	var ret []resNodeData
	for _, val := range rst {
		if _, ok := mp[val.ResId]; ok {
			continue
		}
		ret = append(ret, val)
	}
	return ret, nil
}

// 获取子资源信息
func (this *ResourceModel) dfs(all []ResData, res_id string, rst *[]ResData) {
	for _, val := range all {
		if val.Res_up_id == res_id {
			*rst = append(*rst, val)
			if val.Res_id == val.Res_up_id {
				logger.Error("层级关系错误,不允许上级菜单域当前菜单编码一致,当前菜单编码:", val.Res_id, "上级菜单编码:", val.Res_up_id)
				return
			}
			this.dfs(all, val.Res_id, rst)
		}
	}
}

//
//func (this *ResourceModel) PostThemeInfo(data url.Values) (string, error) {
//
//	theme_id := data.Get("theme_id")
//	res_id := data.Get("res_id")
//	res_url := data.Get("res_url")
//	res_class := data.Get("res_class")
//	res_img := data.Get("res_img")
//	res_bg_color := data.Get("res_bg_color")
//	group_id := data.Get("group_id")
//	sort_id := data.Get("sort_id")
//	res_open_type := data.Get("res_open_type")
//	res_type := data.Get("res_type")
//
//	if !validator.IsWord(res_id) {
//		logger.Error("资源编码必须由1,30位字母或数字组成")
//		return "error_resource_res_id", errors.New("error_resource_res_id")
//	}
//
//	switch res_type {
//	case "0":
//		// 首页主菜单信息
//		if !validator.IsURI(res_url) {
//			logger.Error("菜单路由地址不能为空")
//			return "error_resource_route_uri", errors.New("error_resource_route_uri")
//		}
//
//		if validator.IsEmpty(res_class) {
//			logger.Error("菜单样式类型不能为空")
//			return "error_resource_class_style", errors.New("error_resource_class_style")
//		}
//
//		if !validator.IsURI(res_img) {
//			logger.Error("菜单图标不能为空")
//			return "error_resource_icon", errors.New("error_resource_icon")
//		}
//
//		if !validator.IsNumeric(group_id) {
//			logger.Error("菜单分组信息必须是数字")
//			return "error_resource_group", errors.New("error_resource_group")
//		}
//
//		if !validator.IsNumeric(sort_id) {
//			logger.Error("菜单排序号必须是数字")
//			return "error_resource_sort", errors.New("error_resource_sort")
//		}
//
//		if validator.IsEmpty(res_open_type) {
//			logger.Error("打开方式不能为空")
//			return "error_resource_open_type", errors.New("error_resource_open_type")
//		}
//
//	case "1":
//		// 子系统菜单信息
//		if !validator.IsURI(res_url) {
//			logger.Error("菜单路由地址不能为空")
//			return "error_resource_route_uri", errors.New("error_resource_route_uri")
//		}
//
//		if validator.IsEmpty(res_class) {
//			logger.Error("菜单样式类型不能为空")
//			return "error_resource_class_style", errors.New("error_resource_class_style")
//		}
//
//		if !validator.IsURI(res_img) {
//			logger.Error("菜单图标不能为空")
//			return "error_resource_icon", errors.New("error_resource_icon")
//		}
//
//		if !validator.IsNumeric(group_id) {
//			logger.Error("菜单分组信息必须是数字")
//			return "error_resource_group", errors.New("error_resource_group")
//		}
//
//		if !validator.IsNumeric(sort_id) {
//			logger.Error("菜单排序号必须是数字")
//			return "error_resource_sort", errors.New("error_resource_sort")
//		}
//
//		if validator.IsEmpty(res_open_type) {
//			logger.Error("打开方式不能为空")
//			return "error_resource_open_type", errors.New("error_resource_open_type")
//		}
//
//	case "2":
//		// 功能按钮信息
//		if !validator.IsURI(res_url) {
//			logger.Error("菜单路由地址不能为空")
//			return "error_resource_route_uri", errors.New("error_resource_route_uri")
//		}
//		sort_id = "0"
//		res_img = ""
//		group_id = ""
//		res_class = ""
//		res_bg_color = ""
//		res_open_type = ""
//
//	case "4":
//		return "success", nil
//	default:
//		return "error_resource_type", errors.New("error_resource_type")
//	}
//
//	tx, err := dbobj.Begin()
//	if err != nil {
//		logger.Error(err)
//		return "error_sql_begin", err
//	}
//
//
//	// add sys_theme_value
//	_, err = tx.Exec(sys_rdbms_073, theme_id, res_id, res_url, res_open_type, res_bg_color, res_class, group_id, res_img, sort_id)
//	if err != nil {
//		logger.Error(err)
//		tx.Rollback()
//		return "error_resource_theme_add", err
//	}
//
//	if tx.Commit() != nil {
//		logger.Error(err)
//		return "error_resource_commit", err
//	}
//	return "success", nil
//}
