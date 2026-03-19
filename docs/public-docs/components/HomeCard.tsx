import React from 'react';

interface CardListItemProps {
  number?: string;
  title: string;
  description?: string;
  href: string;
  badge?: {
    text: string;
    variant: 'easy' | 'medium' | 'hard';
  };
}

export function CardListItem({ 
  number, 
  title, 
  description, 
  href, 
  badge 
}: CardListItemProps) {
  return (
    <a href={href} className="card-list-item">
      {number && (
        <div className="card-list-item__number">
          {number}
        </div>
      )}
      <div className="card-list-item__content">
        <div className="card-list-item__header">
          <h4 className="card-list-item__title">{title}</h4>
          {badge && (
            <span className={`card-list-item__badge card-list-item__badge--${badge.variant}`}>
              {badge.text}
            </span>
          )}
        </div>
        {description && (
          <p className="card-list-item__description">{description}</p>
        )}
      </div>
      <div className="card-list-item__arrow">
        <svg width="24" height="24" viewBox="0 0 16 16" fill="none" xmlns="http://www.w3.org/2000/svg">
          <path d="M5 11L11 5M11 5H7M11 5V9" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round"/>
        </svg>
      </div>
    </a>
  );
}

interface CardListProps {
  children: React.ReactNode;
}

export function CardList({ children }: CardListProps) {
  return (
    <div className="card-list">
      {children}
    </div>
  );
}

interface HomeCardProps {
  title: string;
  content?: React.ReactNode
  className?: string;
  footerLink?: {
    text: string;
    href: string;
  };
}

export function HomeCard({ 
  title, 
  content,
  className = '',
  footerLink
}: HomeCardProps) {
  return (
    <div 
      className={`home-card ${className}`}
    >
      
      <div className="home-card__content">
        <h3 className="home-card__title">{title}</h3>
        {content}
      </div>
      
      {footerLink && (
        <div className="home-card__footer">
          <a href={footerLink.href} className="home-card__footer-link">
            {footerLink.text} →
          </a>
        </div>
      )}
    </div>
  );
}

interface HomeCardsProps {
  children: React.ReactNode;
  layout?: 'equal' | 'unequal';
  columns?: string;
  gap?: string;
}

export function HomeCards({ 
  children, 
  layout = 'equal',
  columns,
  gap = '2rem'
}: HomeCardsProps) {
  const getGridColumns = () => {
    if (columns) return columns;
    
    switch (layout) {
      case 'equal': return '1fr 1fr';
      case 'unequal': return '1fr 2fr'; // 2:1 ratio
      default: return '1fr 1fr';
    }
  };

  return (
    <div 
      className={`home-cards home-cards--${layout}`}
      style={{
        display: 'grid',
        gridTemplateColumns: getGridColumns(),
        gap: gap,
      }}
    >
      {children}
    </div>
  );
}
