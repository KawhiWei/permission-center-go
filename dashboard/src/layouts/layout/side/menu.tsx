import { useState } from 'react';
import { useLocation, useNavigate } from "react-router-dom";
import { Menu, MenuValue } from "tdesign-react";
import { DashboardIcon, MenuApplicationIcon, UsergroupIcon, UserIcon, AppIcon } from 'tdesign-icons-react';
import { setPageLoading } from '../../../page-loading';

const { MenuItem, SubMenu } = Menu;

interface IProp {
    collapse: boolean;
}

const MenuComponent = (props: IProp) => {
    const { pathname } = useLocation();
    const navigate = useNavigate();
    const [expanded, setExpanded] = useState<string[]>(() => (
        ['/roles', '/menus', '/user-roles'].includes(pathname) ? ['/permission-center'] : []
    ));

    const navigateWithLoading = async (nextPath: string) => {
        if (nextPath === pathname) {
            return;
        }

        setPageLoading(true);
        navigate(nextPath);
    };

    const onMenuChange = (value: MenuValue) => {
        const nextPath = String(value);
        void navigateWithLoading(nextPath);
    };

    return (
        <Menu
            value={pathname}
            collapsed={props.collapse}
            expanded={expanded}
            onChange={onMenuChange}
            onExpand={(values) => setExpanded(values.map(String))}>
            <MenuItem value="/dashboard" icon={<DashboardIcon />}>仪表盘</MenuItem>
            <MenuItem value="/applications" icon={<AppIcon />}>应用管理</MenuItem>
            <SubMenu value="/permission-center" title="权限中心" icon={<DashboardIcon />}>
                <MenuItem value="/roles" icon={<UsergroupIcon />}>角色管理</MenuItem>
                <MenuItem value="/menus" icon={<MenuApplicationIcon />}>菜单与按钮管理</MenuItem>
                <MenuItem value="/user-roles" icon={<UserIcon />}>用户角色绑定</MenuItem>
            </SubMenu>
        </Menu>
    );
};

export default MenuComponent;
