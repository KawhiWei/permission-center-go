import { useEffect, useState } from 'react';
import { useLocation, useNavigate } from 'react-router-dom';
import { Button, Input, Space, Tooltip } from 'tdesign-react';
import { Fullscreen1Icon, FullscreenExit1Icon, MoonIcon, SearchIcon, SunnyIcon } from 'tdesign-icons-react';
import {
  SERVICE_RESOURCE_CHANGE_EVENT,
  SERVICE_RESOURCE_STORAGE_KEY,
  getStoredServiceResource,
} from '../../../api/permission';

interface PublicHeaderProps {
  theme: 'light' | 'dark';
  onChangeTheme: () => void;
}

const PublicHeader = ({ theme, onChangeTheme }: PublicHeaderProps) => {
    const location = useLocation();
    const navigate = useNavigate();
    const [serviceResource, setServiceResource] = useState(getStoredServiceResource);
    const [isFullscreen, setIsFullscreen] = useState(Boolean(document.fullscreenElement));

    useEffect(() => {
        const syncServiceResource = () => {
            setServiceResource(getStoredServiceResource());
        };
        const handleStorage = (event: StorageEvent) => {
            if (event.key === SERVICE_RESOURCE_STORAGE_KEY) {
                syncServiceResource();
            }
        };

        window.addEventListener(SERVICE_RESOURCE_CHANGE_EVENT, syncServiceResource);
        window.addEventListener('storage', handleStorage);
        return () => {
            window.removeEventListener(SERVICE_RESOURCE_CHANGE_EVENT, syncServiceResource);
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
            setIsFullscreen(false);
            return;
        }
        await document.documentElement.requestFullscreen();
        setIsFullscreen(true);
    };

    const handleSwitchServiceResource = () => {
        const redirect = `${location.pathname}${location.search}${location.hash}`;
        navigate(`/select-service-resource?redirect=${encodeURIComponent(redirect)}`);
    };

    return (
        <div className="layout-header-edit" >
            <Space size="medium">
                <div
                  className="layout-current-service-resource"
                  title={serviceResource ? `当前服务资源：${serviceResource}` : '当前未选择服务资源'}
                >
                  <span className="layout-current-service-resource-label">服务资源</span>
                  <strong className="layout-current-service-resource-value">
                    {serviceResource || '未选择'}
                  </strong>
                </div>
                <Button
                  className="layout-service-resource-switch"
                  variant="text"
                  size="small"
                  onClick={handleSwitchServiceResource}
                >
                  切换服务资源
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
                    className="layout-header-tool-button"
                    variant="text"
                    size="small"
                    shape="circle"
                    type="button"
                    aria-label={theme === 'light' ? '切换到暗色主题' : '切换到浅色主题'}
                    icon={theme === 'light' ? <MoonIcon size="18px" /> : <SunnyIcon size="18px" />}
                    onClick={onChangeTheme}
                  />
                </Tooltip>
                <Tooltip
                  placement="bottom"
                  trigger="hover"
                  content={isFullscreen ? '退出全屏' : '全屏显示'}
                >
                  <Button
                    className="layout-header-tool-button"
                    variant="text"
                    size="small"
                    shape="circle"
                    type="button"
                    aria-label={isFullscreen ? '退出全屏' : '进入全屏'}
                    icon={isFullscreen ? <FullscreenExit1Icon size="18px" /> : <Fullscreen1Icon size="18px" />}
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
