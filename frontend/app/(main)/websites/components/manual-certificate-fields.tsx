'use client';

import { useId } from 'react';
import { useTranslations } from 'next-intl';
import type { UseFormReturn } from 'react-hook-form';
import { Field, FieldError, FieldGroup, FieldLabel } from '@/components/ui/field';
import { Input } from '@/components/ui/input';
import { Textarea } from '@/components/ui/textarea';
import type { ManualImportFormValues } from './schemas';

export function ManualCertificateFields({ form }: { form: UseFormReturn<ManualImportFormValues> }) {
  const t = useTranslations('certificates');
  const prefix = useId();
  return (
    <FieldGroup>
      {(['name', 'remark', 'cert_pem', 'key_pem'] as const).map((name) => {
        const id = `${prefix}-${name}`;
        const error = form.formState.errors[name];
        const props = {
          id,
          'aria-invalid': Boolean(error),
          'aria-describedby': error ? `${id}-error` : undefined,
          ...form.register(name),
        };
        return (
          <Field key={name} data-invalid={Boolean(error)}>
            <FieldLabel htmlFor={id}>{t(name === 'cert_pem' ? 'certPem' : name === 'key_pem' ? 'keyPem' : name)}</FieldLabel>
            {name === 'cert_pem' || name === 'key_pem' ? (
              <Textarea {...props} className='min-h-32 font-mono' autoCapitalize='none' spellCheck={false} />
            ) : (
              <Input {...props} />
            )}
            <FieldError id={`${id}-error`} errors={[error]} />
          </Field>
        );
      })}
    </FieldGroup>
  );
}
