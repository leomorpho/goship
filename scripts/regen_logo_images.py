import os
from PIL import Image
import json

def find_png_files(directory):
    png_files = []
    for root, dirs, files in os.walk(directory):
        for file in files:
            if file.lower().endswith('.png'):
                png_files.append(os.path.join(root, file))
    return png_files

def add_background(image, background_color=(255, 255, 255)):
    # Create a white background image of the same size as the original image
    background = Image.new("RGB", image.size, background_color)
    # Paste the original image on top of the background, using it as a mask to retain transparency
    background.paste(image, mask=image.split()[3])  # Use the alpha channel as a mask
    return background

def update_images(directories, new_image_path, manifest_json_path):
    # Open the new image
    new_image = Image.open(new_image_path)

    # Collect all PNG files from the given directories
    existing_paths = []
    for directory in directories:
        existing_paths.extend(find_png_files(directory))

    for path in existing_paths:
        try:
            # Open the existing image to get its size
            with Image.open(path) as img:
                size = img.size

            # Resize the new image to match the existing image's size
            resized_new_image = new_image.resize(size, Image.LANCZOS)

            # If the file is in the iOS directory, remove alpha transparency
            if 'ios-wrapper' in path:
                resized_new_image = add_background(resized_new_image, (0, 0, 0))

            # Save the resized new image, overwriting the existing image
            resized_new_image.save(path)

            print(f"Updated: {path}")
        except Exception as e:
            print(f"Error updating {path}: {str(e)}")

    # Load the manifest.json file
    with open(manifest_json_path, 'r') as manifest_file:
        manifest_data = json.load(manifest_file)
    
    # Process icons in the manifest.json file
    if 'icons' in manifest_data:
        for icon in manifest_data['icons']:
            src = icon.get('src')
            sizes = icon.get('sizes')

            if src and sizes:
                try:
                    width, height = map(int, sizes.split('x'))
                    icon_size = (width, height)
                    
                    # Resize the new image to match the icon's size
                    resized_new_image = new_image.resize(icon_size, Image.LANCZOS)

                    # Define the path for saving the resized icon
                    icon_path = os.path.join("static", os.path.basename(src))

                    # Save the resized icon
                    resized_new_image.save(icon_path)

                    print(f"Created icon: {icon_path} with size {sizes}")
                except Exception as e:
                    print(f"Error creating icon {src}: {str(e)}")
                    
    print("Image update process completed.")

# List of directories to search for PNG files
directories = [
    "./static",
    "./pwabuilder-android-wrapper",
    "./pwabuilder-ios-wrapper"
]

# Path to the new image
new_image_path = "/Users/leoaudibert/Downloads/goshop.PNG"
manifest_json_path = "./static/manifest.json"

# Run the update process
update_images(directories, new_image_path, manifest_json_path)
