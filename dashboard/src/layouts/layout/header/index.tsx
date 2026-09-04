import { useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Button, Input, Space, Tooltip } from 'tdesign-react';
import { FullscreenExitIcon, FullscreenIcon, MoonIcon, SearchIcon, SunnyIcon } from 'tdesign-icons-react';
import {
  APPLICATION_CHANGE_EVENT,
  APPLICATION_STORAGE_KEY,
  getStoredApplication,
} from '../../../api/permission';

interface PublicHeaderProps {
  theme: 'light' | 'dark';
  onChangeTheme: () => void;
}

const PublicHeader = ({ theme, onChangeTheme }: PublicHeaderProps) => {
    const location = useLocation();
    const navigate = useNavigate();
    const [application, setApplication] = useState(getStoredApplication);
    const [isFullscreen, setIsFullscreen] = useState(Boolean(document.fullscreenElement));

    useEffect(() => {
        const syncApplication = () => {
            setApplication(getStoredApplication());
        };
        const handleStorage = (event: StorageEvent) => {
            if (event.key === APPLICATION_STORAGE_KEY) {
                syncApplication();
            }
        };

        window.addEventListener(APPLICATION_CHANGE_EVENT, syncApplication);
        window.addEventListener('storage', handleStorage);
        return () => {
            window.removeEventListener(APPLICATION_CHANGE_EVENT, syncApplication);
            window.removeEventListener('storage', handleStorage);
        };
    }, []);

    useEffect(() => {
        const handleFullscreenChange = () => {
            setIsFullscreen(Boolean(document.fullscreenElement));
        };

        document.addEventListener('fullscreenchange', handleFullscreenChange);
        return () => {
            document.removeEventListener('fullscreenchange', handleFullscreenChange);
        };
    }, []);

    const handleToggleFullscreen = async () => {
        if (document.fullscreenElement) {
            await document.exitFullscreen();
            return;
        }
        await document.documentElement.requestFullscreen();
    };

    const handleSwitchApplication = () => {
        const redirect = `${location.pathname}${location.search}${location.hash}`;
        navigate(`/select-application?redirect=${encodeURIComponent(redirect)}`);
    };

    return (
        <div className="layout-header-edit" >
            <Space size="medium">
                <div
                  className="layout-current-application"
                  title={application ? `当前应用：${application}` : '当前未选择应用'}
                >
                  <span className="layout-current-application-label">应用</span>
                  <strong className="layout-current-application-value">
                    {application || '未选择'}
                  </strong>
                </div>
                <Button
                  className="layout-application-switch"
                  variant="text"
                  size="small"
                  onClick={handleSwitchApplication}
                >
                  切换应用
                </Button>
                <Input
                  style={{
                    width: 200,
                  }}
                  aria-label="全局搜索"
                  prefixIcon={<SearchIcon />}
                  placeholder="请输入内容查询"
                />
                <Tooltip
                  placement="bottom"
                  trigger="hover"
                  content={`点击切换为${theme === 'light' ? '暗黑' : '亮色'}模式`}
                >
                  <Button
                    size="small"
                    shape="circle"
                    icon={theme === 'light' ? <MoonIcon /> : <SunnyIcon />}
                    onClick={onChangeTheme}
                  />
                </Tooltip>
                <Tooltip
                  placement="bottom"
                  trigger="hover"
                  content={isFullscreen ? '退出全屏' : '全屏显示'}
                >
                  <Button
                    size="small"
                    shape="circle"
                    icon={isFullscreen ? <FullscreenExitIcon /> : <FullscreenIcon />}
                    onClick={() => {
                      void handleToggleFullscreen();
                    }}
                  />
                </Tooltip>
            </Space>
        </div>

    )
}
export default PublicHeader
