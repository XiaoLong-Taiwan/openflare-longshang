'use client';

import { useMutation, useQuery, useQueryClient } from '@tanstack/react-query';
import { useMemo, useState } from 'react';
import { FileKey, Plus, RefreshCw, Trash2 } from 'lucide-react';
import { toast } from 'sonner';

import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogHeader,
  AlertDialogTitle,
} from '@/components/ui/alert-dialog';
import { Button } from '@/components/ui/button';
import {
  Table,
  TableBody,
  TableCell,
  TableHead,
  TableHeader,
  TableRow,
} from '@/components/ui/table';
import { EmptyStateWithBorder } from '@/components/layout/empty';
import { ErrorInline } from '@/components/layout/error';
import { LoadingStateWithBorder } from '@/components/layout/loading';
import type { TlsCertificateItem } from '@/lib/services/openflare';
import { TlsCertificateService } from '@/lib/services/openflare';
import { formatDateTime } from '@/lib/utils';

import { CertificateApplyDialog } from '../websites/components/certificate-apply-dialog';
import { CertificateDetailDialog } from '../websites/components/certificate-detail-dialog';
import { CertificateEditorDialog } from '../websites/components/certificate-editor-dialog';
import { CertificateImportDialog } from '../websites/components/certificate-import-dialog';
import { WebsiteStatusBadge } from '../websites/components/status-badge';
import { useTranslations } from 'next-intl';

import {
  getCertificateStatus,
  getErrorMessage,
} from '../websites/components/website-utils';

const certificatesQueryKey = ['openflare', 'tls-certificates'];

type CertificateApplyMode = 'edit-acme' | 'convert-upload';

export default function CertificatesPage() {
  const t = useTranslations('certificates');
  const tc = useTranslations('common');
  const queryClient = useQueryClient();
  const [importOpen, setImportOpen] = useState(false);
  const [applyOpen, setApplyOpen] = useState(false);
  const [detailOpen, setDetailOpen] = useState(false);
  const [editorOpen, setEditorOpen] = useState(false);
  const [deleteTarget, setDeleteTarget] = useState<TlsCertificateItem | null>(
    null,
  );
  const [selectedCertificateId, setSelectedCertificateId] = useState<
    number | null
  >(null);
  const [applyCertificate, setApplyCertificate] =
    useState<TlsCertificateItem | null>(null);
  const [applyMode, setApplyMode] = useState<CertificateApplyMode>('edit-acme');

  const certificatesQuery = useQuery({
    queryKey: certificatesQueryKey,
    queryFn: () => TlsCertificateService.list(),
  });

  const deleteMutation = useMutation({
    mutationFn: (id: number) => TlsCertificateService.deleteById(id),
    onSuccess: async () => {
      toast.success(t('deleted'));
      setDeleteTarget(null);
      await queryClient.invalidateQueries({ queryKey: certificatesQueryKey });
    },
    onError: (error) => toast.error(getErrorMessage(error, t('requestFailed'))),
  });

  const renewMutation = useMutation({
    mutationFn: (id: number) => TlsCertificateService.renew(id),
    onSuccess: async (cert) => {
      toast.success(t('renewSubmitted', { name: cert.name }));
      await queryClient.invalidateQueries({ queryKey: certificatesQueryKey });
    },
    onError: (error) => toast.error(getErrorMessage(error, t('requestFailed'))),
  });

  const certificates = useMemo(
    () => certificatesQuery.data ?? [],
    [certificatesQuery.data],
  );

  const handleOpenEditor = (certificate: TlsCertificateItem) => {
    if (certificate.provider === 'acme') {
      setApplyMode('edit-acme');
      setApplyCertificate(certificate);
      setApplyOpen(true);
    } else {
      setSelectedCertificateId(certificate.id);
      setEditorOpen(true);
    }
  };

  return (
    <div className='py-6 px-1 space-y-6'>
      <div className='flex flex-col gap-4 sm:flex-row sm:items-center sm:justify-between'>
        <div className='flex items-center gap-2'>
          <FileKey className='size-5 text-primary' />
          <h1 className='text-2xl font-semibold tracking-tight'>
            {t('title')}
          </h1>
        </div>
        <div className='flex flex-wrap gap-2'>
          <Button
            variant='outline'
            size='sm'
            className='h-7 text-xs'
            onClick={() =>
              void queryClient.invalidateQueries({
                queryKey: certificatesQueryKey,
              })
            }
          >
            <RefreshCw className='size-3.5 mr-1' />
            {t('refresh')}
          </Button>
          <Button
            variant='secondary'
            size='sm'
            className='h-7 text-xs'
            onClick={() => setImportOpen(true)}
          >
            {t('importCert')}
          </Button>
          <Button
            size='sm'
            className='h-7 text-xs'
            onClick={() => setApplyOpen(true)}
          >
            <Plus className='size-3.5 mr-1' />
            {t('applyTitle')}
          </Button>
        </div>
      </div>

      <div className='overflow-x-auto rounded-lg border border-dashed'>
        {certificatesQuery.isLoading ? (
          <LoadingStateWithBorder
            icon={FileKey}
            description={t('loadingList')}
          />
        ) : certificatesQuery.isError ? (
          <div className='p-8'>
            <ErrorInline
              message={getErrorMessage(
                certificatesQuery.error,
                t('requestFailed'),
              )}
              onRetry={() => void certificatesQuery.refetch()}
              className='justify-center'
            />
          </div>
        ) : certificates.length === 0 ? (
          <EmptyStateWithBorder icon={FileKey} description={t('emptyList')} />
        ) : (
          <Table>
            <TableHeader>
              <TableRow>
                <TableHead>{t('columns.name')}</TableHead>
                <TableHead>{t('columns.status')}</TableHead>
                <TableHead>{t('notAfter')}</TableHead>
                <TableHead>{t('columns.source')}</TableHead>
                <TableHead>{t('columns.remark')}</TableHead>
                <TableHead className='w-[220px] text-right'>
                  {t('columns.actions')}
                </TableHead>
              </TableRow>
            </TableHeader>
            <TableBody>
              {certificates.map((certificate) => {
                const status = getCertificateStatus(certificate, t);
                return (
                  <TableRow key={certificate.id}>
                    <TableCell className='font-medium'>
                      {certificate.name}
                    </TableCell>
                    <TableCell>
                      <WebsiteStatusBadge
                        label={status.label}
                        tone={status.tone}
                      />
                    </TableCell>
                    <TableCell className='text-sm text-muted-foreground'>
                      {formatDateTime(certificate.not_after)}
                    </TableCell>
                    <TableCell>
                      {certificate.provider === 'acme'
                        ? t('sourceAcme')
                        : t('sourceUpload')}
                    </TableCell>
                    <TableCell className='max-w-[240px] truncate text-sm text-muted-foreground'>
                      {certificate.remark || t('noRemark')}
                    </TableCell>
                    <TableCell className='text-right'>
                      <div className='flex justify-end gap-1'>
                        <Button
                          variant='ghost'
                          size='sm'
                          onClick={() => {
                            setSelectedCertificateId(certificate.id);
                            setDetailOpen(true);
                          }}
                        >
                          {t('view')}
                        </Button>
                        <Button
                          variant='ghost'
                          size='sm'
                          onClick={() => handleOpenEditor(certificate)}
                        >
                          {t('edit')}
                        </Button>
                        {certificate.provider === 'acme' ? (
                          <Button
                            variant='ghost'
                            size='sm'
                            disabled={renewMutation.isPending}
                            onClick={() => renewMutation.mutate(certificate.id)}
                          >
                            {t('renew')}
                          </Button>
                        ) : null}
                        <Button
                          variant='ghost'
                          size='icon'
                          className='text-destructive hover:text-destructive'
                          title={t('deleteCert')}
                          aria-label={t('deleteCert')}
                          onClick={() => setDeleteTarget(certificate)}
                        >
                          <Trash2 />
                        </Button>
                      </div>
                    </TableCell>
                  </TableRow>
                );
              })}
            </TableBody>
          </Table>
        )}
      </div>

      <CertificateImportDialog
        open={importOpen}
        onOpenChange={setImportOpen}
        onImported={(certificate) =>
          toast.success(t('imported', { name: certificate.name }))
        }
      />

      <CertificateApplyDialog
        open={applyOpen && !applyCertificate}
        onOpenChange={setApplyOpen}
        onApplied={(certificate) =>
          toast.success(t('applySubmitted', { name: certificate.name }))
        }
      />

      {applyCertificate ? (
        <CertificateApplyDialog
          open={applyOpen}
          onOpenChange={(open) => {
            setApplyOpen(open);
            if (!open) setApplyCertificate(null);
          }}
          mode={applyMode}
          certificate={applyCertificate}
          onApplied={(certificate) => {
            setApplyCertificate(null);
            toast.success(
              applyMode === 'convert-upload'
                ? t('convertSubmitted', { name: certificate.name })
                : t('reapplySubmitted', { name: certificate.name }),
            );
          }}
        />
      ) : null}

      <CertificateDetailDialog
        certificateId={selectedCertificateId}
        open={detailOpen}
        onOpenChange={setDetailOpen}
        onEdit={() => {
          setDetailOpen(false);
          const item = certificates.find((c) => c.id === selectedCertificateId);
          if (item) handleOpenEditor(item);
        }}
        onDelete={() => {
          const item = certificates.find((c) => c.id === selectedCertificateId);
          if (item) {
            setDetailOpen(false);
            setDeleteTarget(item);
          }
        }}
        deleting={deleteMutation.isPending}
      />

      <CertificateEditorDialog
        certificateId={selectedCertificateId}
        open={editorOpen}
        onOpenChange={setEditorOpen}
        onSaved={(certificate) =>
          toast.success(t('updated', { name: certificate.name }))
        }
        onConvert={(certificate) => {
          setEditorOpen(false);
          setApplyMode('convert-upload');
          setApplyCertificate(certificate);
          setApplyOpen(true);
        }}
      />

      <AlertDialog
        open={deleteTarget !== null}
        onOpenChange={(open) => !open && setDeleteTarget(null)}
      >
        <AlertDialogContent>
          <AlertDialogHeader>
            <AlertDialogTitle>{t('deleteTitle')}</AlertDialogTitle>
            <AlertDialogDescription>
              {t('deleteDesc', { name: deleteTarget?.name ?? '' })}
            </AlertDialogDescription>
          </AlertDialogHeader>
          <AlertDialogFooter>
            <AlertDialogCancel>{tc('cancel')}</AlertDialogCancel>
            <AlertDialogAction
              className='bg-destructive text-destructive-foreground hover:bg-destructive/90'
              onClick={() =>
                deleteTarget && deleteMutation.mutate(deleteTarget.id)
              }
            >
              {tc('delete')}
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}
