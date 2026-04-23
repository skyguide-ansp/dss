package cleanup

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/interuss/dss/pkg/logging"
	dssmodels "github.com/interuss/dss/pkg/models"
	ridmodels "github.com/interuss/dss/pkg/rid/models"
	ridrepos "github.com/interuss/dss/pkg/rid/repos"
	rids "github.com/interuss/dss/pkg/rid/store"
	scdmodels "github.com/interuss/dss/pkg/scd/models"
	scdrepos "github.com/interuss/dss/pkg/scd/repos"
	scds "github.com/interuss/dss/pkg/scd/store"
	survmodels "github.com/interuss/dss/pkg/surveillance/models"
	survrepos "github.com/interuss/dss/pkg/surveillance/repos"
	survs "github.com/interuss/dss/pkg/surveillance/store"
	"github.com/interuss/stacktrace"
	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

var (
	EvictCmd = &cobra.Command{
		Use:   "evict",
		Short: "List and evict expired entities",
		RunE:  evict,
	}
	flags         = pflag.NewFlagSet("evict", pflag.ExitOnError)
	scdCheckOirs  = flags.Bool("scd_oir", true, "set this flag to true to check for expired SCD operational intents")
	scdCheckSubs  = flags.Bool("scd_sub", true, "set this flag to true to check for expired SCD subscriptions")
	scdTtl        = flags.Duration("scd_ttl", time.Hour*24*112, "time-to-live duration used for determining SCD entries expiration, defaults to 2*56 days")
	ridCheckISAs  = flags.Bool("rid_isa", true, "set this flag to true to check for expired RID ISAs")
	ridCheckSubs  = flags.Bool("rid_sub", true, "set this flag to true to check for expired RID subscriptions")
	ridTtl        = flags.Duration("rid_ttl", time.Minute*30, "time-to-live duration used for determining RID entries expiration, defaults to 30 minutes")
	survCheckTSAs = flags.Bool("surveillance_tsa", true, "set this flag to true to check for expired surveillance TSAs")
	survCheckSubs = flags.Bool("surveillance_sub", true, "set this flag to true to check for expired surveillance subscriptions")
	survTtl       = flags.Duration("surveillance_ttl", time.Minute*30, "time-to-live duration used for determining surveillance entries expiration, defaults to 30 minutes")
	deleteExpired = flags.Bool("delete", false, "set this flag to true to delete the expired entities")
	locality      = flags.String("locality", "", "self-identification string of this DSS instance")
	timeout       = flags.Duration("timeout", 5*time.Minute, "Timeout for the command")
)

func init() {
	EvictCmd.Flags().AddFlagSet(flags)
}

func evict(cmd *cobra.Command, _ []string) error {
	var (
		ctx           = cmd.Context()
		scdThreshold  = time.Now().Add(-*scdTtl)
		ridThreshold  = time.Now().Add(-*ridTtl)
		survThreshold = time.Now().Add(-*survTtl)
	)
	log.Printf("WARNING: The usage of this tool may have an impact on performance when deleting entities. Read more in the README.")

	ctx, cancel := context.WithTimeout(ctx, *timeout)
	defer cancel()

	logger := logging.WithValuesFromContext(ctx, logging.Logger)

	scdStore, err := scds.Init(ctx, logger, false, false)
	if err != nil {
		return err
	}

	ridStore, err := rids.Init(ctx, logger, false)
	if err != nil {
		return err
	}

	survStore, err := survs.Init(ctx, logger, false)
	if err != nil {
		return err
	}

	var (
		scdExpiredOpIntents []*scdmodels.OperationalIntent
		scdExpiredSubs      []*scdmodels.Subscription
		ridExpiredISAs      []*ridmodels.IdentificationServiceArea
		ridExpiredSubs      []*ridmodels.Subscription
		survExpiredTSAs     []*survmodels.TrafficSurveilledArea
		survExpiredSubs     []*survmodels.Subscription
	)
	scdAction := func(ctx context.Context, r scdrepos.Repository) (err error) {
		if *scdCheckOirs {
			scdExpiredOpIntents, err = r.ListExpiredOperationalIntents(ctx, scdThreshold)
			if err != nil {
				return fmt.Errorf("listing expired operational intents: %w", err)
			}
			if *deleteExpired {
				for _, opIntent := range scdExpiredOpIntents {
					if err = r.DeleteOperationalIntent(ctx, opIntent.ID); err != nil {
						return fmt.Errorf("deleting expired operational intents: %w", err)
					}
				}
			}
		}

		if *scdCheckSubs {
			scdExpiredSubs, err = r.ListExpiredSubscriptions(ctx, scdThreshold)
			if err != nil {
				return fmt.Errorf("SCD listing expired subscriptions: %w", err)
			}
			if *deleteExpired {
				for _, sub := range scdExpiredSubs {
					if err = r.DeleteSubscription(ctx, sub.ID); err != nil {
						return fmt.Errorf("SCD deleting expired subscriptions: %w", err)
					}
				}
			}
		}
		return nil
	}
	if err = scdStore.Transact(ctx, scdAction); err != nil {
		return fmt.Errorf("failed to execute SCD transaction: %w", err)
	}

	ridAction := func(ctx context.Context, r ridrepos.Repository) (err error) {
		if *ridCheckISAs {
			ridExpiredISAs, err = r.ListExpiredISAs(ctx, *locality, ridThreshold)
			if err != nil {
				return stacktrace.Propagate(err, "Failed to list expired ISAs")
			}

			if *deleteExpired {
				for _, isa := range ridExpiredISAs {
					_, err := r.DeleteISA(ctx, isa)
					if err != nil {
						return stacktrace.Propagate(err, "Failed to delete ISAs")
					}
				}
			}

		}

		if *ridCheckSubs {
			ridExpiredSubs, err = r.ListExpiredSubscriptions(ctx, *locality, ridThreshold)
			if err != nil {
				return stacktrace.Propagate(err,
					"Failed to list RID expired Subscriptions")
			}

			if *deleteExpired {
				for _, sub := range ridExpiredSubs {
					_, err := r.DeleteSubscription(ctx, sub)
					if err != nil {
						return stacktrace.Propagate(err, "Failed to delete RID Subscription")
					}
				}
			}

		}

		return nil
	}
	if err = ridStore.Transact(ctx, ridAction); err != nil {
		return fmt.Errorf("failed to execute RID transaction: %w", err)
	}

	survAction := func(ctx context.Context, r survrepos.Repository) (err error) {
		if *survCheckTSAs {

			survExpiredTSAs, err = r.ListExpiredISAs(ctx, *locality, survThreshold)
			if err != nil {
				return stacktrace.Propagate(err, "Failed to list expired TSAs")
			}

			if *deleteExpired {
				for _, tsa := range survExpiredTSAs {
					_, err := r.DeleteISA(ctx, tsa)
					if err != nil {
						return stacktrace.Propagate(err, "Failed to delete TSAs")
					}
				}
			}

		}

		if *survCheckSubs {
			survExpiredSubs, err = r.ListExpiredSubscriptions(ctx, *locality, survThreshold)
			if err != nil {
				return stacktrace.Propagate(err,
					"Failed to list RID expired Subscriptions")
			}

			if *deleteExpired {
				for _, sub := range survExpiredSubs {
					_, err := r.DeleteSubscription(ctx, sub)
					if err != nil {
						return stacktrace.Propagate(err, "Failed to delete Surveillance Subscription")
					}
				}
			}

		}

		return nil
	}
	if err = survStore.Transact(ctx, survAction); err != nil {
		return fmt.Errorf("failed to execute Surveillance transaction: %w", err)
	}

	for _, opIntent := range scdExpiredOpIntents {
		logExpiredEntity("operational intent", opIntent.ID, scdThreshold, *deleteExpired, opIntent.EndTime != nil)
	}
	for _, sub := range scdExpiredSubs {
		logExpiredEntity("SCD subscription", sub.ID, scdThreshold, *deleteExpired, sub.EndTime != nil)
	}
	if len(scdExpiredOpIntents)+len(scdExpiredSubs) == 0 {
		log.Printf("no SCD entity older than %s found", scdThreshold.String())
	}

	for _, isa := range ridExpiredISAs {
		logExpiredEntity("ISA", isa.ID, ridThreshold, *deleteExpired, isa.EndTime != nil)
	}
	for _, sub := range ridExpiredSubs {
		logExpiredEntity("RID subscription", sub.ID, ridThreshold, *deleteExpired, sub.EndTime != nil)
	}
	if len(ridExpiredISAs)+len(ridExpiredSubs) == 0 {
		log.Printf("no RID entity older than %s found", ridThreshold.String())
	}

	for _, tsa := range survExpiredTSAs {
		logExpiredEntity("TSA", tsa.ID, survThreshold, *deleteExpired, tsa.EndTime != nil)
	}
	for _, sub := range survExpiredSubs {
		logExpiredEntity("surveillance subscription", sub.ID, survThreshold, *deleteExpired, sub.EndTime != nil)
	}
	if len(survExpiredTSAs)+len(survExpiredSubs) == 0 {
		log.Printf("no surveillance entity older than %s found", survThreshold.String())
	}

	if !*deleteExpired {
		log.Printf("no entity was deleted, run the command again with the `--delete` flag to do so")
	}
	return nil
}

func logExpiredEntity(entity string, entityID dssmodels.ID, threshold time.Time, deleted, hasEndTime bool) {
	logMsg := "found"
	if deleted {
		logMsg = "deleted"
	}

	expMsg := "last update before %s (missing end time)"
	if hasEndTime {
		expMsg = "end time before %s"
	}
	log.Printf("%s %s %s; expired due to %s", logMsg, entity, entityID.String(), fmt.Sprintf(expMsg, threshold.String()))
}
