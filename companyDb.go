package main

import (
	"context"
	"fmt"
	"strings"

	"cloud.google.com/go/firestore"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

func userExists(ctx context.Context, client *firestore.Client, userId string) (bool, error) {
	if strings.TrimSpace(userId) == "" {
		return false, fmt.Errorf("no user id")
	}

	docs, err := client.Collection("users").Documents(ctx).GetAll()
	if err != nil {
		logError.Printf("Failed to retrieve documents: %v", err)
	}

	for _, doc := range docs {
		fmt.Println("Document ID:", doc.Ref.ID)
		fmt.Println("Document Data:", doc.Data())
	}

	_, err = client.Collection("users").Doc(userId).Get(ctx)
	if err != nil && status.Code(err) == codes.NotFound {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func addUser(ctx context.Context, client *firestore.Client, userId string) error {
	if exists, err := userExists(ctx, client, userId); err != nil {
		return err
	} else if exists {
		return nil
	}
	_, err := client.Collection("users").Doc(userId).Set(ctx, map[string]any{})
	if err != nil {
		logError.Printf("Failed to create user: %v Err %v", userId, err)
		return fmt.Errorf("failed to create user: %v", userId)
	}
	return nil
}

func deleteCollection(ctx context.Context, client *firestore.Client, collection *firestore.CollectionRef) error {
	iter := collection.Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return fmt.Errorf("failed to get documents in collection %v: %w", collection.Path, err)
	}
	bw := client.BulkWriter(ctx)
	for _, doc := range docs {
		if _, err = bw.Delete(doc.Ref); err != nil {
			return fmt.Errorf("failed to delete document %v: %w", doc.Ref.Path, err)
		}
	}
	bw.End()
	return nil
}

func deleteCompanies(ctx context.Context, client *firestore.Client, userId string) error {
	if exists, err := userExists(ctx, client, userId); err != nil {
		return err
	} else if !exists {
		return fmt.Errorf("user does not exist: %v", userId)
	}
	userDocRef := client.Collection("users").Doc(userId)
	deleteCollection(ctx, client, userDocRef.Collection("companyDetails"))
	return nil
}

func deleteUser(ctx context.Context, client *firestore.Client, userId string) error {
	if err := deleteCompanies(ctx, client, userId); err != nil {
		return err
	}

	// Add code here to delete other subcollections as needed

	// Delete the user document
	userDocRef := client.Collection("users").Doc(userId)
	_, err := userDocRef.Delete(ctx)
	if err != nil {
		return fmt.Errorf("failed to delete user: %v", userId)
	}

	return nil
}

type companyDetails struct {
	ID      string `firestore:"id,omitempty"`
	Name    string `firestore:"name,omitempty"`
	Address string `firestore:"address,omitempty"`
}

func getCompanies(ctx context.Context, client *firestore.Client, userId string) ([]companyDetails, error) {

	if exists, err := userExists(ctx, client, userId); err != nil {
		return nil, err
	} else if !exists {
		return nil, fmt.Errorf("user does not exist: %v", userId)
	}
	iter := client.Collection("users").Doc(userId).Collection("companyDetails").Documents(ctx)
	docs, err := iter.GetAll()
	if err != nil {
		return nil, err
	}
	clinics := make([]companyDetails, len(docs))
	for i, doc := range docs {
		var clinic companyDetails
		err := doc.DataTo(&clinic)
		clinic.ID = doc.Ref.ID
		if err != nil {
			return nil, err
		}
		clinics[i] = clinic
	}

	return clinics, nil
}

// add the user if it does not already exist
// Update the list with the new company list
// Remove the elements that are not in the list
func setCompanies(ctx context.Context, client *firestore.Client, userId string, companyList []companyDetails) error {
	if len(companyList) == 0 {
		deleteCompanies(ctx, client, userId)
		return nil
	}

	addUser(ctx, client, userId)
	currentCompanies, err := getCompanies(ctx, client, userId)
	if err != nil {
		return err
	}
	retainMap := make(map[string]string)
	for _, clinic := range companyList {
		retainMap[clinic.ID] = clinic.ID
	}
	deleteItems := []string{}
	for _, clinic := range currentCompanies {
		if _, ok := retainMap[clinic.ID]; !ok {
			deleteItems = append(deleteItems, clinic.ID)
		}
	}

	companyDetails := client.Collection("users").Doc(userId).Collection("companyDetails")
	bw := client.BulkWriter(ctx)
	for _, clinicId := range deleteItems {
		docRef := companyDetails.Doc(clinicId)
		_, err := bw.Delete(docRef)
		if err != nil {
			return err
		}
	}
	var docRef *firestore.DocumentRef
	for _, clinic := range companyList {
		if clinic.ID == "" {
			docRef = companyDetails.NewDoc()
		} else {
			docRef = companyDetails.Doc(clinic.ID)
		}
		doc := map[string]interface{}{
			"name":    clinic.Name,
			"address": clinic.Address,
		}
		_, err := bw.Set(docRef, doc, firestore.MergeAll)
		if err != nil {
			return err
		}
	}
	bw.End()
	return nil
}
