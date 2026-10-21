"use client";

import { useEffect, useState } from "react";
import type { FormEvent } from "react";
import { apiFetch, errMsg } from "@/lib/api";
import type { Review } from "@/lib/types";
import { formatDateTime } from "@/lib/format";
import { useAuthStore } from "@/store/auth";
import {
  Button,
  Card,
  EmptyState,
  ErrorBox,
  Field,
  SkeletonText,
  SuccessBox,
  TextArea,
} from "@/components/ui";

function Stars({
  value,
  onPick,
}: {
  value: number;
  onPick?: (v: number) => void;
}) {
  return (
    <div className="flex items-center gap-1">
      {[1, 2, 3, 4, 5].map((s) => (
        <button
          key={s}
          type="button"
          disabled={!onPick}
          onClick={() => onPick?.(s)}
          aria-label={`${s} bintang`}
          className={`text-2xl leading-none ${
            s <= value ? "text-amber-400" : "text-zinc-300"
          } ${onPick ? "cursor-pointer hover:scale-110" : ""} transition`}
        >
          ★
        </button>
      ))}
    </div>
  );
}

function ReviewForm({
  eventId,
  existing,
  onSaved,
}: {
  eventId: number;
  existing: Review | null;
  onSaved: () => void;
}) {
  const [rating, setRating] = useState(existing?.rating ?? 5);
  const [text, setText] = useState(existing?.text ?? "");
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState("");
  const [success, setSuccess] = useState("");

  const submit = async (e: FormEvent) => {
    e.preventDefault();
    setSaving(true);
    setError("");
    setSuccess("");
    try {
      if (existing) {
        await apiFetch(`/api/events/${eventId}/reviews`, {
          method: "PUT",
          body: JSON.stringify({ rating, text: text || undefined }),
        });
        setSuccess("Ulasan diperbarui.");
      } else {
        await apiFetch(`/api/events/${eventId}/reviews`, {
          method: "POST",
          body: JSON.stringify({ rating, text: text || undefined }),
        });
        setSuccess("Terima kasih atas ulasanmu!");
      }
      onSaved();
    } catch (err) {
      setError(errMsg(err));
    } finally {
      setSaving(false);
    }
  };

  return (
    <form onSubmit={submit} className="space-y-3 rounded-lg border border-zinc-200 bg-zinc-50 p-4">
      <ErrorBox message={error} />
      <SuccessBox message={success} />
      <Field label="Ratingmu">
        <Stars value={rating} onPick={setRating} />
      </Field>
      <Field label="Ceritakan pengalamanmu (opsional)">
        <TextArea
          value={text}
          onChange={(e) => setText(e.target.value)}
          placeholder="Bagaimana acaranya?"
        />
      </Field>
      <Button disabled={saving}>
        {saving ? "Menyimpan..." : existing ? "Perbarui Ulasan" : "Kirim Ulasan"}
      </Button>
    </form>
  );
}

/**
 * Rating & ulasan event: menampilkan bintang + jumlah, daftar ulasan publik,
 * dan form beri/edit ulasan (hanya untuk user login; 422 backend bila tak
 * eligible ditampilkan apa adanya).
 */
export default function ReviewSection({
  eventId,
  ratingAvg,
  ratingCount,
}: {
  eventId: number;
  ratingAvg?: number | null;
  ratingCount?: number | null;
}) {
  const { token, user } = useAuthStore();
  const [reviews, setReviews] = useState<Review[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [formKey, setFormKey] = useState(0);

  const load = () => {
    setLoading(true);
    apiFetch<{ reviews: Review[]; avg?: number; count?: number }>(
      `/api/events/${eventId}/reviews`
    )
      .then((d) => setReviews(d.reviews))
      .catch((e) => setError(errMsg(e)))
      .finally(() => setLoading(false));
  };

  useEffect(() => {
    void Promise.resolve().then(load);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [eventId]);

  const myReview = user
    ? reviews.find((r) => r.user_id === user.id) ?? null
    : null;

  return (
    <Card>
      <h2 className="mb-3 font-semibold text-zinc-900">Ulasan</h2>

      <div className="mb-4 flex items-center gap-3">
        {ratingAvg != null && ratingAvg > 0 ? (
          <>
            <span className="text-3xl font-bold text-zinc-900">
              {ratingAvg.toFixed(1)}
            </span>
            <div>
              <Stars value={Math.round(ratingAvg)} />
              <p className="mt-1 text-xs text-zinc-500">
                {ratingCount ?? reviews.length} ulasan
              </p>
            </div>
          </>
        ) : (
          <p className="text-sm text-zinc-500">Belum ada ulasan untuk event ini.</p>
        )}
      </div>

      <ErrorBox message={error} />
      {loading ? (
        <SkeletonText lines={3} />
      ) : reviews.length > 0 ? (
        <div className="mb-6 space-y-4">
          {reviews.map((r) => (
            <div key={r.id} className="border-b border-zinc-100 pb-4 last:border-0">
              <div className="flex items-center justify-between gap-2">
                <div className="flex items-center gap-2">
                  <Stars value={r.rating} />
                  <span className="text-sm font-medium text-zinc-900">
                    {r.user_name ?? "Pengunjung"}
                  </span>
                  {user && r.user_id === user.id && (
                    <span className="rounded-full bg-orange-100 px-2 py-0.5 text-xs font-medium text-orange-700">
                      Ulasanmu
                    </span>
                  )}
                </div>
                <span className="text-xs text-zinc-400">
                  {formatDateTime(r.created_at)}
                </span>
              </div>
              {r.text && (
                <p className="mt-1.5 whitespace-pre-line text-sm text-zinc-600">
                  {r.text}
                </p>
              )}
            </div>
          ))}
        </div>
      ) : (
        !loading && <EmptyState message="Jadilah yang pertama memberi ulasan." />
      )}

      {token && (
        <div className="mt-4">
          <h3 className="mb-2 text-sm font-semibold text-zinc-900">
            {myReview ? "Edit ulasanmu" : "Beri ulasan"}
          </h3>
          <ReviewForm
            key={formKey}
            eventId={eventId}
            existing={myReview}
            onSaved={() => {
              load();
              setFormKey((k) => k + 1);
            }}
          />
        </div>
      )}
    </Card>
  );
}
