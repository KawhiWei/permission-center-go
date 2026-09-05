import './style.less';

import { Layout, Tabs } from 'tdesign-react';
import { ChevronLeftIcon, ChevronRightIcon } from 'tdesign-icons-react';
import { applyThemeMode, getThemeMode } from '../../theme';
import { getPageLoading, subscribePageLoading } from '../../page-loading';
import { useEffect, useMemo, useState, useSyncExternalStore } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';

import AvatarComponent from './avatar';
import GlobalLoading from '../../components/global-loading';
import LogoComponent from './logo';
import PublicContent from './content';
import PublicHeader from './header';
import SliderMenu from './side';
import {
  findNavigationMenu,
  useDynamicNavigation,
  type NavigationMenu,
} from '../../router/dynamic-routes';
import { DEFAULT_LAYOUT_TAB, getStoredLayoutTabs, LAYOUT_TABS_STORAGE_KEY, type LayoutTabItem } from './tab-storage';

const { Content, Aside, Header } = Layout;
const { TabPanel } = Tabs;

const DEFAULT_TAB_PATH = DEFAULT_LAYOUT_TAB.value;
const DEFAULT_TAB_LABEL = DEFAULT_LAYOUT_TAB.label;

const navigationPaths = (menus: NavigationMenu[]): string[] => (
  menus.flatMap((menu) => [menu.path, ...navigationPaths(menu.children)])
);

const PublicLayout = () => {
  const { pathname } = useLocation();
  const navigate = useNavigate();
  const pageLoading = useSyncExternalStore(subscribePageLoading, getPageLoading, getPageLoading);
  const { menus, loading: navigationLoading, version: navigationVersion } = useDynamicNavigation();

  const [collapsed, setCollapsed] = useState(false);
  const [theme, setTheme] = useState<'light' | 'dark'>(() => getThemeMode());
  const [tabs, setTabs] = useState<LayoutTabItem[]>(getStoredLayoutTabs);

  useEffect(() => {
    window.localStorage.setItem(LAYOUT_TABS_STORAGE_KEY, JSON.stringify(tabs));
  }, [tabs]);

  const defaultTabLabel = useMemo(
    () => menus.find((menu) => menu.path === DEFAULT_TAB_PATH)?.name || DEFAULT_TAB_LABEL,
    [menus],
  );

  const currentTabLabel = useMemo(() => {
    if (pathname === DEFAULT_TAB_PATH) {
      return defaultTabLabel;
    }

    return findNavigationMenu(menus, pathname)?.name || pathname;
  }, [defaultTabLabel, menus, pathname]);

  useEffect(() => {
    if (navigationLoading) {
      return;
    }

    setTabs((prev) => {
      const existing = prev.find((tab) => tab.value === pathname);
      if (existing) {
        return prev.map((tab) => (
          tab.value === pathname
            ? { ...tab, label: currentTabLabel, removable: pathname !== DEFAULT_TAB_PATH }
            : tab
        ));
      }

      const nextTab: LayoutTabItem = {
        value: pathname,
        label: currentTabLabel,
        removable: pathname !== DEFAULT_TAB_PATH,
      };

      const dashboardTab = prev.find((tab) => tab.value === DEFAULT_TAB_PATH) || {
        value: DEFAULT_TAB_PATH,
        label: defaultTabLabel,
        removable: false,
      };
      const restTabs = prev.filter((tab) => tab.value !== DEFAULT_TAB_PATH);

      if (pathname === DEFAULT_TAB_PATH) {
        return [dashboardTab, ...restTabs];
      }

      return [dashboardTab, ...restTabs, nextTab];
    });
  }, [currentTabLabel, defaultTabLabel, navigationLoading, pathname]);

  useEffect(() => {
    if (navigationLoading) {
      setTabs([{ ...DEFAULT_LAYOUT_TAB }]);
      return;
    }

    const validPaths = new Set([DEFAULT_TAB_PATH, ...navigationPaths(menus)]);
    setTabs((prev) => {
      const next = prev.filter((tab) => validPaths.has(tab.value));
      return next.length > 0 ? next : [{ ...DEFAULT_LAYOUT_TAB, label: defaultTabLabel }];
    });
  }, [defaultTabLabel, menus, navigationLoading, navigationVersion]);

  useEffect(() => {
    if (currentTabLabel) {
      document.title = currentTabLabel;
    }
  }, [currentTabLabel, pathname])

  const handleCollapsed = () => {
    setCollapsed((prev) => !prev);
  };

  const handleChangeTheme = () => {
    const next = theme === 'light' ? 'dark' : 'light';
    setTheme(next);
    applyThemeMode(next);
  };

  const handleTabChange = (value: string | number) => {
    if (value === undefined || value === null) {
      return;
    }

    const nextPath = String(value);
    if (!nextPath || nextPath === 'undefined' || nextPath === pathname) {
      return;
    }

    navigate(nextPath);
  };

  const handleTabRemove = ({ value }: { value: string | number }) => {
    if (value === undefined || value === null) {
      return;
    }

    const closingKey = String(value);
    if (!closingKey || closingKey === 'undefined') {
      return;
    }

    const closingIndex = tabs.findIndex((tab) => tab.value === closingKey);
    if (closingIndex === -1) {
      return;
    }

    const nextTabs = tabs.filter((tab) => tab.value !== closingKey);
    setTabs(nextTabs);

    if (pathname !== closingKey) {
      return;
    }

    const fallbackTab = nextTabs[closingIndex] || nextTabs[closingIndex - 1] || nextTabs[0];
    const fallbackPath = fallbackTab?.value || DEFAULT_TAB_PATH;
    if (fallbackPath && fallbackPath !== pathname) {
      navigate(fallbackPath);
    }
  };

  return (

    <Layout className="layout-container">
      {(pageLoading || navigationLoading) && <GlobalLoading />}
      <Header className="layout-header">
        <div className="layout-header-left">
          <LogoComponent collapse={false} />
        </div>
        <div className="layout-header-right">
          <PublicHeader theme={theme} onChangeTheme={handleChangeTheme} />
          <AvatarComponent />
        </div>
      </Header>

      <Layout className="layout-body">
        <Aside
          width={collapsed ? '64px' : '240px'}
          className={`layout-sider${collapsed ? ' is-collapsed' : ''}`}
        >
          <div className="layout-sider-menu">
            <SliderMenu collapse={collapsed} menus={menus} />
          </div>
          <div className="layout-sider-trigger-bottom" onClick={handleCollapsed}>
            {collapsed ? <ChevronRightIcon /> : <ChevronLeftIcon />}
          </div>
        </Aside>

        <Layout className="layout-main">
          <Content className="layout-content">
            <div className="layout-content-tabs">
              <Tabs
                value={pathname}
                size="medium"
                theme="normal"
                scrollPosition="auto"
                onChange={handleTabChange}
                onRemove={handleTabRemove}
              >
                {tabs.map((tab) => (
                  <TabPanel
                    key={tab.value}
                    value={tab.value}
                    label={tab.label}
                    removable={tab.removable}
                  />
                ))}
              </Tabs>
            </div>
            <PublicContent menus={menus} navigationLoading={navigationLoading} />
          </Content>
        </Layout>
      </Layout>
    </Layout>)
}
export default PublicLayout
