import './style.less';

interface BrandComponentProps {
  className?: string;
  compact?: boolean;
}

const BrandComponent = ({ className, compact = false }: BrandComponentProps) => {
  const classes = ['brand', compact ? 'brand--compact' : '', className || '']
    .filter(Boolean)
    .join(' ');

  return (
    <div className={classes} aria-label="NexusAuth 权限中心">
      <img
        className="brand__mark"
        src="/brand/nexusauth-logo.svg"
        alt={compact ? 'NexusAuth' : ''}
      />
      {!compact && (
        <span className="brand__copy">
          <span className="brand__name">NexusAuth</span>
          <span className="brand__product">权限中心</span>
        </span>
      )}
    </div>
  );
};

export default BrandComponent;
