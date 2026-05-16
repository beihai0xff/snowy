import React from 'react';

export type Subject = 'physics' | 'biology' | 'chemistry' | 'math';

const LABEL: Record<Subject, string> = {
  physics: '物理',
  biology: '生物',
  chemistry: '化学',
  math: '数学',
};

const CLS: Record<Subject, string> = {
  physics: 'snowy-subject-badge--physics',
  biology: 'snowy-subject-badge--biology',
  chemistry: 'snowy-subject-badge--chem',
  math: 'snowy-subject-badge--math',
};

export default function SubjectBadge({ subject, label }: { subject: Subject; label?: string }) {
  return <span className={`snowy-subject-badge ${CLS[subject]}`}>{label ?? LABEL[subject]}</span>;
}
