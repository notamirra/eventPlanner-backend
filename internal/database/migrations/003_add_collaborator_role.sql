-- Add collaborator role to participant_role enum
DO $$
BEGIN
    IF NOT EXISTS (
        SELECT 1 FROM pg_type 
        WHERE typname = 'participant_role' 
        AND EXISTS (
            SELECT 1 FROM pg_enum 
            WHERE enumlabel = 'collaborator' 
            AND enumtypid = (SELECT oid FROM pg_type WHERE typname = 'participant_role')
        )
    ) THEN
        ALTER TYPE participant_role ADD VALUE 'collaborator';
    END IF;
END $$;
