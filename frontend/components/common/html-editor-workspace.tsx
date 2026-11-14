'use client';

import { useRef, useState } from 'react';
import { useTheme } from 'next-themes';
import CodeMirror from '@uiw/react-codemirror';
import { html } from '@codemirror/lang-html';
import { Expand, Terminal } from 'lucide-react';
import Link from 'next/link';
import { useTranslations } from 'next-intl';

import { Button } from '@/components/ui/button';
import { Tabs, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { cn } from '@/lib/utils';
import {
  ORIGIN_ERROR_PAGE_HTML_MAX_BYTES,
  previewOriginErrorPageHTML,
} from '@/lib/openflare/default-origin-error-page-html';

type HtmlEditorWorkspaceProps = {
  value: string;
  onChange: (value: string) => void;
  toolbarRight?: React.ReactNode;
  maxBytes?: number;
  preview?: (html: string) => string;
  footerHint?: React.ReactNode;
  showPreviewLink?: boolean;
  previewTitle?: string;
};

/**
 * 通用 HTML 编辑器工作区：左代码 / 右实时预览横向布局，可拖拽分隔。
 */
export function HtmlEditorWorkspace({
  value,
  onChange,
  toolbarRight,
  maxBytes = ORIGIN_ERROR_PAGE_HTML_MAX_BYTES,
  preview = previewOriginErrorPageHTML,
  footerHint = (
    <>
      {'{{status}}'}→400 · {'{{host}}'}→example.com
    </>
  ),
  showPreviewLink = true,
  previewTitle,
}: HtmlEditorWorkspaceProps) {
  const t = useTranslations('htmlWorkspace');
  const containerRef = useRef<HTMLDivElement>(null);
  const [mobileView, setMobileView] = useState('editor');
  const [editorWidthPercent, setEditorWidthPercent] = useState(48);
  const { resolvedTheme } = useTheme();
  const cmTheme = resolvedTheme === 'dark' ? 'dark' : 'light';
  const previewSrcDoc = preview(value);
  const htmlBytes = new TextEncoder().encode(value).length;

  const resizeAt = (clientX: number) => {
    const container = containerRef.current;
    if (!container) return;
    const rect = container.getBoundingClientRect();
    if (rect.width > 0) {
      setEditorWidthPercent(Math.min(75, Math.max(25, ((clientX - rect.left) / rect.width) * 100)));
    }
  };

  return (
    <div
      ref={containerRef}
      style={{ '--editor-width': `${editorWidthPercent}%` } as React.CSSProperties}
      className='w-full min-w-0 border border-border/40 bg-card/60 backdrop-blur-md rounded-lg overflow-hidden flex flex-col shadow-sm h-[calc(100dvh-160px)] min-h-[360px] md:min-h-[520px]'
    >
      <div className='flex items-center justify-between px-4 py-2 border-b bg-muted/40 shrink-0 gap-4 flex-wrap'>
        <div className='flex items-center gap-2 min-w-0'>
          <Terminal className='size-4 text-primary shrink-0' />
          <span className='text-xs font-semibold'>{t('editor')}</span>
          <span className='text-[11px] text-muted-foreground font-mono truncate'>
            {t('bytes', { used: htmlBytes, max: maxBytes })}
            {value.trim() === '' ? ` · ${t('emptyHint')}` : ''}
          </span>
        </div>
        <div className='flex items-center gap-1.5 flex-wrap'>
          {toolbarRight}
        </div>
      </div>

      <Tabs value={mobileView} onValueChange={setMobileView} className='shrink-0 p-2 md:hidden'>
        <TabsList className='w-full' aria-label={t('view')}>
          <TabsTrigger value='editor'>{t('editor')}</TabsTrigger>
          <TabsTrigger value='preview'>{t('livePreview')}</TabsTrigger>
        </TabsList>
      </Tabs>
      <div className='flex flex-1 min-h-0 overflow-hidden'>
        <div
          className={cn('h-full w-full md:w-(--editor-width) overflow-hidden relative min-w-0 bg-background md:border-r border-border/40', mobileView === 'editor' ? 'block' : 'hidden md:block')}
        >
          <CodeMirror
            value={value}
            height='100%'
            extensions={[html()]}
            theme={cmTheme}
            onChange={onChange}
            className='h-full text-xs font-mono'
            basicSetup={{
              lineNumbers: true,
              foldGutter: true,
              dropCursor: true,
              allowMultipleSelections: false,
              indentOnInput: true,
            }}
            placeholder={t('placeholder')}
          />
        </div>

        <div
          role='separator'
          tabIndex={0}
          aria-label={t('resize')}
          aria-orientation='vertical'
          aria-valuemin={25}
          aria-valuemax={75}
          aria-valuenow={Math.round(editorWidthPercent)}
          onPointerDown={(event) => {
            if (event.button !== 0) return;
            event.preventDefault();
            event.currentTarget.setPointerCapture(event.pointerId);
            resizeAt(event.clientX);
          }}
          onPointerMove={(event) => {
            if (event.currentTarget.hasPointerCapture(event.pointerId)) resizeAt(event.clientX);
          }}
          onPointerUp={(event) => {
            if (event.currentTarget.hasPointerCapture(event.pointerId)) event.currentTarget.releasePointerCapture(event.pointerId);
          }}
          onKeyDown={(event) => {
            if (!['ArrowLeft', 'ArrowRight', 'Home', 'End'].includes(event.key)) return;
            event.preventDefault();
            setEditorWidthPercent((width) => event.key === 'Home' ? 25 : event.key === 'End' ? 75 : Math.min(75, Math.max(25, width + (event.key === 'ArrowLeft' ? -2 : 2))));
          }}
          className='hidden md:flex w-3 touch-none bg-border/60 hover:bg-primary/50 focus-visible:outline-2 focus-visible:outline-ring cursor-col-resize transition-colors items-center justify-center shrink-0 select-none'
          title={t('resize')}
        >
          <div className='h-8 w-1 rounded bg-muted-foreground/30' />
        </div>

        <div className={cn('flex-1 min-w-0 flex-col overflow-hidden bg-muted/10', mobileView === 'preview' ? 'flex' : 'hidden md:flex')}>
          <div className='flex items-center justify-between px-4 py-1.5 border-b bg-muted/20 shrink-0 text-[11px] text-muted-foreground font-mono gap-2'>
            <span className='font-semibold'>{t('livePreview')}</span>
            <div className='flex items-center gap-2'>
              <span
                className={footerHint === null ? 'hidden' : 'hidden sm:inline'}
              >
                {footerHint ?? null}
              </span>
              {showPreviewLink ? (
                <Button
                  variant='ghost'
                  size='sm'
                  className='h-6 px-2 text-[11px]'
                  asChild
                >
                  <Link href='/responses/error-page/preview'>
                    <Expand className='size-3' />
                    {t('preview')}
                  </Link>
                </Button>
              ) : null}
            </div>
          </div>
          <div className='flex-1 min-h-0 overflow-hidden bg-background'>
            <iframe
              title={previewTitle ?? t('previewTitle')}
              sandbox=''
              srcDoc={previewSrcDoc}
              className='h-full w-full border-0 bg-background'
            />
          </div>
        </div>
      </div>
    </div>
  );
}
