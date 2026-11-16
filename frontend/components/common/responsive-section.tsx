'use client';

import { type ReactNode, useState } from 'react';
import { useCompactLayout } from '@/hooks/use-compact-layout';
import { ChevronDown } from 'lucide-react';
import { Button } from '@/components/ui/button';
import {
  Collapsible,
  CollapsibleContent,
  CollapsibleTrigger,
} from '@/components/ui/collapsible';

export function ResponsiveSection({
  title,
  children,
  defaultOpen = false,
}: {
  title: string;
  children: ReactNode;
  defaultOpen?: boolean;
}) {
  const [open, setOpen] = useState(defaultOpen);
  const compact = useCompactLayout();
  return (
    <Collapsible open={open} onOpenChange={setOpen} className='min-w-0'>
      <CollapsibleTrigger asChild>
        <Button
          variant='outline'
          className='group mb-3 h-auto min-h-11 w-full justify-between whitespace-normal text-left lg:hidden'
        >
          {title}
          <ChevronDown
            data-icon='inline-end'
            className='shrink-0 group-data-[state=open]:rotate-180'
          />
        </Button>
      </CollapsibleTrigger>
      <CollapsibleContent
        forceMount
        className='hidden min-w-0 data-[state=open]:block lg:block'
      >
        {!compact || open ? children : null}
      </CollapsibleContent>
    </Collapsible>
  );
}
