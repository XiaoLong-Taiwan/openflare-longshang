'use client';

import { zodResolver } from '@hookform/resolvers/zod';
import { useMutation, useQueryClient } from '@tanstack/react-query';
import { type FormEvent, useEffect, useState } from 'react';
import { useForm } from 'react-hook-form';
import { Loader2 } from 'lucide-react';

import { Button } from '@/components/ui/button';
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogHeader,
  DialogTitle,
} from '@/components/ui/dialog';
import { Input } from '@/components/ui/input';
import { Label } from '@/components/ui/label';
import { Tabs, TabsContent, TabsList, TabsTrigger } from '@/components/ui/tabs';
import { ManualCertificateFields } from './manual-certificate-fields';
import type { TlsCertificateItem } from '@/lib/services/openflare';
import { TlsCertificateService } from '@/lib/services/openflare';

import { useTranslations } from 'next-intl';

import {
  createManualImportSchema,
  defaultFileImportValues,
  defaultManualImportValues,
  type FileImportFormValues,
  type ManualImportFormValues,
} from './schemas';
import {
  getErrorMessage,
  toFilePayload,
  toManualPayload,
} from './website-utils';

const certificatesQueryKey = ['openflare', 'tls-certificates'];

interface CertificateImportDialogProps {
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onImported?: (certificate: TlsCertificateItem) => void;
}

export function CertificateImportDialog({
  open,
  onOpenChange,
  onImported,
}: CertificateImportDialogProps) {
  const t = useTranslations('certificates');
  const queryClient = useQueryClient();
  const [importMode, setImportMode] = useState<'manual' | 'file'>('manual');
  const [error, setError] = useState('');
  const [fileForm, setFileForm] = useState<FileImportFormValues>(
    defaultFileImportValues,
  );
  const [certFile, setCertFile] = useState<File | null>(null);
  const [keyFile, setKeyFile] = useState<File | null>(null);
  const [fileInputNonce, setFileInputNonce] = useState(0);

  const manualForm = useForm<ManualImportFormValues>({
    resolver: zodResolver(createManualImportSchema(t)),
    defaultValues: defaultManualImportValues,
  });

  useEffect(() => {
    if (!open) return;
    setError('');
  }, [open]);

  const invalidateQueries = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: certificatesQueryKey }),
    ]);
  };

  const resetFileForm = () => {
    setFileForm(defaultFileImportValues);
    setCertFile(null);
    setKeyFile(null);
    setFileInputNonce((value) => value + 1);
  };

  const resetAll = () => {
    setImportMode('manual');
    setError('');
    manualForm.reset(defaultManualImportValues);
    resetFileForm();
  };

  const handleClose = () => {
    resetAll();
    onOpenChange(false);
  };

  const manualImportMutation = useMutation({
    mutationFn: (values: ManualImportFormValues) =>
      TlsCertificateService.create(toManualPayload(values)),
    onSuccess: async (certificate) => {
      await invalidateQueries();
      onImported?.(certificate);
      handleClose();
    },
    onError: (err) => setError(getErrorMessage(err, t('requestFailed'))),
  });

  const fileImportMutation = useMutation({
    mutationFn: (values: FileImportFormValues) =>
      TlsCertificateService.importFile(
        toFilePayload(values, certFile, keyFile, t('selectFiles')),
      ),
    onSuccess: async (certificate) => {
      await invalidateQueries();
      onImported?.(certificate);
      handleClose();
    },
    onError: (err) => setError(getErrorMessage(err, t('requestFailed'))),
  });

  const handleManualSubmit = manualForm.handleSubmit((values) => {
    setError('');
    manualImportMutation.mutate(values);
  });

  const handleFileSubmit = (event: FormEvent<HTMLFormElement>) => {
    event.preventDefault();
    setError('');
    fileImportMutation.mutate(fileForm);
  };

  const pending =
    manualImportMutation.isPending || fileImportMutation.isPending;

  return (
    <Dialog open={open} onOpenChange={(next) => !next && handleClose()}>
      <DialogContent className='max-w-2xl max-h-[90dvh] overflow-y-auto'>
        <DialogHeader>
          <DialogTitle>{t('importTitle')}</DialogTitle>
          <DialogDescription>{t('importDesc')}</DialogDescription>
        </DialogHeader>

        {error ? <p role='alert' className='text-sm text-destructive'>{error}</p> : null}

        <Tabs
          value={importMode}
          onValueChange={(value) => setImportMode(value as 'manual' | 'file')}
        >
          <TabsList className='grid w-full grid-cols-2'>
            <TabsTrigger value='manual'>{t('manualImport')}</TabsTrigger>
            <TabsTrigger value='file'>{t('fileImport')}</TabsTrigger>
          </TabsList>

          <TabsContent value='manual' className='space-y-4'>
            <form className='space-y-4' onSubmit={handleManualSubmit}>
              <ManualCertificateFields form={manualForm} />

              <Button type='submit' disabled={pending}>
                {pending ? (
                  <>
                    <Loader2 className='mr-1 size-3.5 animate-spin' />
                    {t('importing')}
                  </>
                ) : (
                  t('importCert')
                )}
              </Button>
            </form>
          </TabsContent>

          <TabsContent value='file' forceMount hidden={importMode !== 'file'} className='space-y-4'>
            <form className='space-y-4' onSubmit={handleFileSubmit}>
              <div className='grid gap-4 md:grid-cols-2'>
                <div className='space-y-2'>
                  <Label htmlFor='certificate-file-name'>{t('name')}</Label>
                  <Input
                    id='certificate-file-name'
                    required
                    maxLength={255}
                    value={fileForm.name}
                    onChange={(event) =>
                      setFileForm((current) => ({
                        ...current,
                        name: event.target.value,
                      }))
                    }
                    placeholder='wildcard-example'
                  />
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='certificate-file-remark'>{t('remark')}</Label>
                  <Input
                    id='certificate-file-remark'
                    maxLength={255}
                    value={fileForm.remark}
                    onChange={(event) =>
                      setFileForm((current) => ({
                        ...current,
                        remark: event.target.value,
                      }))
                    }
                    placeholder={t('remarkWildcardPlaceholder')}
                  />
                </div>
              </div>

              <div className='grid gap-4 md:grid-cols-2'>
                <div className='space-y-2'>
                  <Label htmlFor='certificate-cert-file'>{t('certFile')}</Label>
                  <Input
                    id='certificate-cert-file'
                    required
                    key={`cert-${fileInputNonce}`}
                    type='file'
                    accept='.pem,.crt,.cer'
                    onChange={(event) =>
                      setCertFile(event.target.files?.[0] ?? null)
                    }
                  />
                  <p className='text-xs text-muted-foreground'>
                    {certFile
                      ? t('fileSelected', { name: certFile.name })
                      : t('selectCertFile')}
                  </p>
                </div>
                <div className='space-y-2'>
                  <Label htmlFor='certificate-key-file'>{t('keyFile')}</Label>
                  <Input
                    id='certificate-key-file'
                    required
                    key={`key-${fileInputNonce}`}
                    type='file'
                    accept='.key,.pem'
                    onChange={(event) =>
                      setKeyFile(event.target.files?.[0] ?? null)
                    }
                  />
                  <p className='text-xs text-muted-foreground'>
                    {keyFile
                      ? t('fileSelected', { name: keyFile.name })
                      : t('selectKeyFile')}
                  </p>
                </div>
              </div>

              <div className='flex gap-2'>
                <Button type='submit' disabled={pending}>
                  {pending ? (
                    <>
                      <Loader2 className='mr-1 size-3.5 animate-spin' />
                      {t('uploading')}
                    </>
                  ) : (
                    t('uploadFiles')
                  )}
                </Button>
                <Button
                  type='button'
                  variant='outline'
                  disabled={pending}
                  onClick={resetFileForm}
                >
                  {t('clearFiles')}
                </Button>
              </div>
            </form>
          </TabsContent>
        </Tabs>
      </DialogContent>
    </Dialog>
  );
}
